# budget_me — CI/CD (GitHub Actions → VPS)

This is the **primary** deploy path. A push to `main` tests and builds the Docker
images of the apps that changed on GitHub Actions and pushes them to the GitHub
Container Registry (GHCR). The deploy then SSHes into the VPS and rolls the
release out **blue-green** (see §6); a failed deploy puts traffic and schema back
by itself and reds the run. Rollback of either app is the manual **Rollback**
workflow (§5). Learning lives on the same VPS at `learning.<domain>` with its own
colours, database and releases log. The VPS never builds images. The workflows
are in [`.github/workflows/`](../../.github/workflows/) (`ci.yml` tests and builds,
`deploy.yml` is the reusable deploy, `rollback.yml` the rollback); the manual
fallback lives in [`DEPLOY.md`](DEPLOY.md).

```
push main ─▶ test + build (only what changed)     (GitHub-hosted runner)
                  │
                  └─▶ push to ghcr.io/kstartfd/budget_me
                                    │
                                    ▼
                  deploy (budget and/or learning, one lock for the whole server)
                    ssh deploy@45.133.19.54
                    render ~/budget_me/.env[.learning]  ·  docker login
                    bluegreen.sh <app> deploy:
                      backup → migrate up → start idle colour
                      → health → caddy reload → watch → release log
                  Rollback workflow  (Actions → Rollback → Run workflow)
```

- Repository: `github.com/kstartfd/budget_me`
- Registry: `ghcr.io/kstartfd/budget_me`
  (`…/backend:<sha>`, `…/frontend:<sha>`, plus a moving `:latest`; learning's are
  `…/learning-backend:<sha>` and `…/learning-frontend:<sha>`)
- Server: `deploy@45.133.19.54` (Ubuntu 26.04, Docker + Compose v2, 4 GB RAM /
  2 CPU / 50 GB, Toronto). Root bootstrap → `deploy` user; see §2.
- Domain: `budgetme.xyz` — DNS at hoster.by, apex `A` record → the VPS IP.

---

## 1. One-time GitHub setup

### 1a. Packages
The build jobs push to GHCR with the workflow's own `GITHUB_TOKEN`
(`permissions: packages: write` in `ci.yml`); nothing to enable. Packages are
created private on the first push.

### 1b. A token for the server to pull images
GitHub → Settings → Developer settings → **Personal access tokens** → generate a
token with the single scope **`read:packages`**. Its owner and value become the
secrets `GHCR_PULL_USER` / `GHCR_PULL_TOKEN`; the deploy job runs
`docker login ghcr.io` with them on the server.

### 1c. Create the SSH deploy key pair (on your machine)
```bash
ssh-keygen -t ed25519 -C budget_me-deploy -f ~/.ssh/budget_me_deploy -N ''
```
- **Public** key `~/.ssh/budget_me_deploy.pub` → append to the server's
  `~deploy/.ssh/authorized_keys` (see §2).
- **Private** key `~/.ssh/budget_me_deploy` → the `SSH_PRIVATE_KEY` secret.

Get the server host keys for `SSH_KNOWN_HOSTS`:
```bash
ssh-keyscan 45.133.19.54
```
Copy the whole output (all lines).

### 1d. Add the secrets
Settings → Secrets and variables → **Actions**. The full list, per app, is in the
header of [`deploy.yml`](../../.github/workflows/deploy.yml). The deploy jobs run
in the GitHub **environments** `production` (budget) and `learning`, so the
secrets can live there instead of at repository level.

| Secret                  | Value                                          |
|-------------------------|------------------------------------------------|
| `SSH_PRIVATE_KEY`       | contents of `~/.ssh/budget_me_deploy`          |
| `SSH_HOST`              | `45.133.19.54`                                 |
| `SSH_KNOWN_HOSTS`       | output of `ssh-keyscan 45.133.19.54`           |
| `GHCR_PULL_USER`        | GitHub user that owns the token from §1b       |
| `GHCR_PULL_TOKEN`       | the `read:packages` token from §1b             |
| `DOMAIN`                | `budgetme.xyz`                                 |
| `POSTGRES_PASSWORD`     | `openssl rand -hex 24`                         |
| `AUTH_SECRET`           | `openssl rand -hex 32`                         |
| `TELEGRAM_BOT_TOKEN`    | @BotFather token (may start empty)             |
| `ACCOUNT_POSTGRES_PASSWORD` | `openssl rand -hex 24` (role `account_app`, shared by both apps) |
| `LEARNING_DOMAIN`       | optional — defaults to `learning.budgetme.xyz` |
| `LEARNING_POSTGRES_PASSWORD` | `openssl rand -hex 24` (role `learning_app`) |
| `LEARNING_AUTH_SECRET`  | `openssl rand -hex 32`                          |
| `LEARNING_TELEGRAM_BOT_TOKEN` | token of the learning bot                |

`ACCOUNT_POSTGRES_PASSWORD` is the one value both apps share: the password of the
`account_app` role that owns the shared account database
([`libs/account`](../../libs/account/README.md)). CI renders it into both apps'
env files; if you keep it in the `.local` files instead it must be the **same**
value in both, because each deploy re-applies it to the role.

Each deploy refuses to start (before touching anything live) if a required value
is empty — budget: `DOMAIN`, `POSTGRES_PASSWORD`, `ACCOUNT_POSTGRES_PASSWORD`,
`AUTH_SECRET`; learning: the same plus `TELEGRAM_BOT_TOKEN`. Each app's secrets
may instead live on the server — `~/budget_me/.env.local` (budget) and
`~/budget_me/.env.learning.local` (learning), `KEY=value` lines, `chmod 600`:
CI wins when it has a value, the file fills whatever CI left empty
(`scripts/merge-env.sh`). The `LEARNING_*` secrets map onto learning's
`DOMAIN`, `POSTGRES_PASSWORD`, `AUTH_SECRET`, `TELEGRAM_BOT_TOKEN`; the same names
in the `.local` file are plain, without the prefix.

`GITHUB_TOKEN`, `GITHUB_SHA` and `GITHUB_REPOSITORY` are provided by GitHub
automatically — do not add them.

> Keep secrets shell-safe (hex / alphanumeric). The deploy job injects them into
> a heredoc and an `ssh` command; a literal `"`, `$`, backtick or `'` could break
> quoting. `openssl rand -hex` and Telegram tokens are safe.

---

## 2. One-time server bootstrap (as `root`)

Nothing to build and no repo to clone — the deploy job ships the compose file,
Caddyfile and a rendered `.env`. The provider hands you a root-only box, so the
bootstrap creates the `deploy` user the workflow logs in as:

```bash
# Docker CE from Docker's apt repo (suite `resolute`), pinned to the versions
# prod runs — never "latest": 29.8.0 / containerd 2.3.5 / buildx 0.37.0 / compose 5.5.1
apt-get install -y docker-ce=5:29.8.0-1~ubuntu.26.04~resolute \
  docker-ce-cli=5:29.8.0-1~ubuntu.26.04~resolute containerd.io=2.3.5-1~ubuntu.26.04~resolute \
  docker-buildx-plugin=0.37.0-1~ubuntu.26.04~resolute docker-compose-plugin=5.5.1-1~ubuntu.26.04~resolute

# deploy user + docker access
useradd -m -s /bin/bash deploy
usermod -aG docker deploy

# authorize the CI deploy key (contents of ~/.ssh/budget_me_deploy.pub)
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
echo 'ssh-ed25519 AAAA…  github-actions' > /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys && chown -R deploy:deploy /home/deploy/.ssh

# directories the workflow writes into (compose + infra/edge/Caddyfile + scripts)
sudo -u deploy mkdir -p /home/deploy/budget_me/infra/edge /home/deploy/budget_me/scripts

# firewall — only SSH + HTTP/HTTPS (Postgres is never published)
apt-get install -y ufw
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw --force enable
```

**Swap is not needed** — image builds happen in CI, not here.

**DNS:** point the apex `A` record `budgetme.xyz` at the VPS IP (at hoster.by →
Домены → budgetme.xyz → Управление → DNS) and wait for it to resolve
(`dig +short budgetme.xyz @u1.hoster.by`) before the first deploy, or Caddy's
first Let's Encrypt challenge will fail and retry. Caddy serves only the apex —
no `www` record needed.

---

## 3. First release

1. Push the branch to `main`:
   ```bash
   git push origin fresh-budget_me:main
   ```
2. Watch the run (Actions → CI). The build jobs push `:<sha>` and `:latest` to
   GHCR; **deploy** then runs automatically.
3. The deploy job renders `~/budget_me/.env`, `docker login`s, then runs
   `scripts/bluegreen.sh budget deploy` (§6): migrate, idle colour, health gate,
   Caddy switch, automatic switch-back on failure.
4. Open `https://budgetme.xyz`. First hit may lag a few seconds while Caddy
   fetches the TLS cert — follow it with
   `ssh deploy@45.133.19.54 'cd ~/budget_me && docker compose -f prod.yml logs -f caddy'`.

### Where the deploy job puts files (and how compose reads them)
The `./infra/edge/Caddyfile` bind-mount in `prod.yml` resolves against the
compose file's own directory, so the layout on the server is:

```
~/budget_me/
├── prod.yml                     # scp of infra/budget/compose/prod.yml (host stack + budget colours)
├── learning.yml                 # scp of infra/learning/compose/prod.yml (learning colours)
├── .env / .env.learning         # rendered from CI secrets + blue-green state, chmod 600
├── releases-budget.log          # one line per successful deploy — what rollbacks read
├── releases-learning.log
├── scripts/bluegreen.sh         # deploy / rollback / status
└── infra/edge/                  # → mounted read-only at /etc/caddy
    ├── Caddyfile                # scp of infra/edge/Caddyfile (two site blocks)
    ├── budget.active.caddy      # live colour of budget — owned by bluegreen.sh
    └── learning.active.caddy    # live colour of learning — owned by bluegreen.sh
```

The job runs `cd ~/budget_me && docker compose --env-file .env -f prod.yml …`, so
the mount resolves to `~/budget_me/infra/edge/` — exactly where the job placed
the Caddyfile and where bluegreen.sh writes the `*.active.caddy` files. The
deploy never copies the repo's `*.active.caddy` over the server's: that would
silently switch colours.

### Category presets (automatic)
The 6 shared presets are seeded by migration `0004_seed_categories.sql`
(idempotent), which the deploy job's migrate step applies — nothing to do by hand.

### Register the Telegram Mini App
In @BotFather (`/newapp` or Bot Settings → Mini App) set the Web App URL to
`https://budgetme.xyz`. Put the bot token into the `TELEGRAM_BOT_TOKEN` CI
secret and re-run deploy so the backend picks it up.

---

## 4. Updating (every later release)

```
push main → tests + builds of what changed go green → deploy runs by itself
```

A push to `main` deploys exactly the commit it built, and only the app(s) whose
sources changed (the shared account module and the workflows count for both), so
deploying one app never drags the other along. Deploys of both apps share one
lock and queue on the VPS. See §6 for what the deploy does on the box.

## 5. Rollback

Actions → **Rollback** → *Run workflow*: pick the app (`budget` or `learning`) and
how many deploys to go back (`steps`):

| `steps` | Goes to | How |
| --- | --- | --- |
| `1` (default) | the previous successful deploy | usually still running in the idle colour → health + Caddy switch, seconds, no pull |
| `2` | two successful deploys back | idle colour redeployed from the registry → health → switch |

Steps count successful deploys back from the last one (`releases-<app>.log`);
rollbacks never write that log, so running `2` twice changes nothing the
second time. After the switch the schema is migrated **down** to the version the
target release was deployed with, by the newer image (only it has the Down SQL),
after a local backup. To keep the schema and the data written since, tick
`keep_schema`.

On the box: `./scripts/bluegreen.sh budget rollback 2`, `… status`.

## 6. Blue-green on the box

```
            ┌───────────── caddy :80/:443 ─────────────┐
            │ budgetme.xyz          learning.budgetme.xyz │
            │ import budget.active  import learning.active│
            └──────┬──────────────────────────┬──────────┘
       backend-blue │ backend-green   learning-backend-blue │ learning-backend-green
      frontend-blue │ frontend-green learning-frontend-blue │ learning-frontend-green
                    └──────────── postgres ────────────┘
                     db budget_me        db learning_app (role learning_app)
```

`scripts/bluegreen.sh <app> deploy <backend-image> <frontend-image>`:

0. the shared `account_app` role and database are created if missing (both apps'
   deploys do it, idempotently);
1. local backup of the app's database;
2. `migrate up` with the **new** image (before the switch: the live colour keeps
   working on the expanded schema — see CODING_STYLE.md, Migrations; it also
   brings the account schema up), then `migrate backfill-accounts`, which links
   the app's users to their accounts;
3. start the **idle** colour, check it really runs the new image, health-check
   backend `/healthz` and frontend `/` over the internal network;
4. rewrite `<app>.active.caddy`, `caddy reload` — zero downtime;
5. poll `https://<domain>/healthz` through Caddy for ~60s; three failures in a
   row switch traffic back;
6. append the release to `releases-<app>.log`.

A failure in 2–5 puts traffic back on the old colour and runs
`migrate down-to <previous version>`, and the run goes red. The old colour
stays up after a good deploy: it is the instant the `steps=1` rollback.

The first blue-green deploy of budget migrates the old in-place stack: traffic
stays on the old `backend`/`frontend` services until blue is healthy, Caddy is
recreated once with the directory mount (a second or two), then the old
containers are removed.
