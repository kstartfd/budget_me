# budget_me — CI/CD (GitLab → VPS)

This is the **primary** deploy path. A budget change pushed to `main` builds the backend and
frontend Docker images on the GitLab runner, pushes them to the GitLab Container
Registry, and the **deploy:budget** job SSHes into the VPS and rolls the release
out **blue-green** (see §6). Learning lives on the same VPS at
`learning.<domain>` with its own colours and database and its own
**deploy:learning** job (off until `LEARNING_DEPLOY_ENABLED=true`).
Every green `main` pipeline that changes budget files auto-deploys (no manual gate). The VPS never builds
images. The pipeline is defined in [`.gitlab-ci.yml`](../../.gitlab-ci.yml); the
manual fallback lives in [`DEPLOY.md`](DEPLOY.md).

```
push main ─▶ build:backend ─┐            (Docker-in-Docker on the runner)
            build:frontend ─┴─▶ push to registry.gitlab.com/kstartfd/budget_me
                                    │
                                    ▼
                              deploy:budget (auto): ssh deploy@45.133.19.54
                                render ~/budget_me/.env  ·  docker login
                                bluegreen.sh budget deploy:
                                  backup → migrate up → start idle colour
                                  → health → caddy reload → watch → release log
                              rollback:budget:1 / :2 (manual buttons)
```

- Project: `gitlab.com/kstartfd/budget_me`
- Registry: `registry.gitlab.com/kstartfd/budget_me`
  (`…/backend:<sha>`, `…/frontend:<sha>`, plus a moving `:latest`)
- Server: `deploy@45.133.19.54` (Ubuntu 26.04, Docker + Compose v2, 4 GB RAM /
  2 CPU / 50 GB, Toronto). Root bootstrap → `deploy` user; see §2.
- Domain: `budgetme.xyz` — DNS at hoster.by, apex `A` record → the VPS IP.

---

## 1. One-time GitLab setup

### 1a. Enable the Container Registry
Settings → General → Visibility, project features, permissions → make sure
**Container Registry** is **On**. (The build jobs push here; without it they 403.)

### 1b. Create a Deploy Token for the server to pull images
Settings → Repository → **Deploy tokens** → *Add token*:
- Name: `vps-pull` (anything)
- Scopes: **`read_registry`** only
- Create, then copy the generated **username** and **token** — shown once.

These become the CI variables `DEPLOY_TOKEN_USER` / `DEPLOY_TOKEN_PASSWORD`; the
deploy job runs `docker login registry.gitlab.com` with them on the server.

### 1c. Create the SSH deploy key pair (on your machine)
```bash
ssh-keygen -t ed25519 -C budget_me-deploy -f ~/.ssh/budget_me_deploy -N ''
```
- **Public** key `~/.ssh/budget_me_deploy.pub` → append to the server's
  `~deploy/.ssh/authorized_keys` (see §2).
- **Private** key `~/.ssh/budget_me_deploy` → the `SSH_PRIVATE_KEY` CI variable.

Get the server host keys for `SSH_KNOWN_HOSTS`:
```bash
ssh-keyscan 45.133.19.54
```
Copy the whole output (all lines).

### 1d. Add the CI/CD variables
Settings → CI/CD → **Variables**. For each: set **Protected** (only protected
refs — `main`/tags — run these jobs) and **Masked** where marked. `SSH_PRIVATE_KEY`
is a multi-line key, so it **cannot be masked** — set its **Type = File** instead.

| Variable                | Value                                          | Type     | Protected | Masked |
|-------------------------|------------------------------------------------|----------|-----------|--------|
| `SSH_PRIVATE_KEY`       | contents of `~/.ssh/budget_me_deploy`          | **File** | yes       | —      |
| `SSH_HOST`              | `45.133.19.54`                                 | Variable | yes       | no     |
| `SSH_KNOWN_HOSTS`       | output of `ssh-keyscan 45.133.19.54`           | Variable | yes       | no     |
| `DEPLOY_TOKEN_USER`     | deploy-token username (from §1b)               | Variable | yes       | no     |
| `DEPLOY_TOKEN_PASSWORD` | deploy-token secret (from §1b)                 | Variable | yes       | **yes**|
| `DOMAIN`                | `budgetme.xyz`                                 | Variable | yes       | no     |
| `POSTGRES_PASSWORD`     | `openssl rand -hex 24`                         | Variable | yes       | **yes**|
| `AUTH_SECRET`           | `openssl rand -hex 32`                          | Variable | yes       | **yes**|
| `TELEGRAM_BOT_TOKEN`    | @BotFather token (may start empty)             | Variable | yes       | **yes**|
| `LEARNING_DOMAIN`       | `learning.budgetme.xyz` (default if empty)     | Variable | yes       | no     |
| `LEARNING_DEPLOY_ENABLED` | `true` to enable deploy/rollback of learning | Variable | yes       | no     |
| `LEARNING_POSTGRES_PASSWORD` | `openssl rand -hex 24` (role `learning_app`) | Variable | yes  | **yes**|
| `LEARNING_AUTH_SECRET`  | `openssl rand -hex 32`                          | Variable | yes       | **yes**|
| `LEARNING_TELEGRAM_BOT_TOKEN` | token of the learning bot                | Variable | yes       | **yes**|

`CI_REGISTRY`, `CI_REGISTRY_USER`, `CI_REGISTRY_PASSWORD`, `CI_REGISTRY_IMAGE`,
`CI_COMMIT_SHORT_SHA` are provided by GitLab automatically — do not add them.

> Keep secrets shell-safe (hex / alphanumeric). The deploy job injects them into
> a heredoc and an `ssh` command; a literal `"`, `$`, backtick or `'` could break
> quoting. `openssl rand -hex` and Telegram tokens are safe.

---

## 2. One-time server bootstrap (as `root`)

Nothing to build and no repo to clone — the deploy job ships the compose file,
Caddyfile and a rendered `.env`. The provider hands you a root-only box, so the
bootstrap creates the `deploy` user the pipeline logs in as:

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
echo 'ssh-ed25519 AAAA…  gitlab-ci' > /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys && chown -R deploy:deploy /home/deploy/.ssh

# directories the pipeline writes into (compose + infra/edge/Caddyfile + scripts)
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
2. Watch the pipeline (CI/CD → Pipelines). `build:backend` + `build:frontend`
   build and push `:<sha>` and `:latest`; **deploy** then runs automatically.
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
├── .env / .env.learning         # rendered from CI variables + blue-green state, chmod 600
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
variable and re-run deploy so the backend picks it up.

---

## 4. Updating (every later release)

```
push main → test + build:* go green → deploy:budget runs automatically
```

Every green `main` pipeline that changes budget files auto-deploys (no
`when: manual`). See §6 for what the deploy does on the box.

## 5. Rollback

Two manual buttons on every `main` pipeline (CI/CD → Pipelines → the pipeline →
stage *rollback*):

| Job | Goes to | How |
| --- | --- | --- |
| `rollback:budget:1` | the previous successful deploy | usually still running in the idle colour → health + Caddy switch, seconds, no pull |
| `rollback:budget:2` | two successful deploys back | idle colour redeployed from the registry → health → switch |

Steps count successful deploys back from the last one (`releases-budget.log`);
rollbacks never write that log, so running `:2` twice changes nothing the
second time. After the switch the schema is migrated **down** to the version the
target release was deployed with, by the newer image (only it has the Down SQL),
after a local backup. To keep the schema and the data written since, run the job
with the variable `ROLLBACK_KEEP_SCHEMA=1` (Run job → Variables). Learning has
the same pair, `rollback:learning:1/2`.

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

1. local backup of the app's database;
2. `migrate up` with the **new** image (before the switch: the live colour keeps
   working on the expanded schema — see CODING_STYLE.md, Migrations);
3. start the **idle** colour, check it really runs the new image, health-check
   backend `/healthz` and frontend `/` over the internal network;
4. rewrite `<app>.active.caddy`, `caddy reload` — zero downtime;
5. poll `https://<domain>/healthz` through Caddy for ~60s; three failures in a
   row switch traffic back;
6. append the release to `releases-<app>.log`.

A failure in 2–5 puts traffic back on the old colour and runs
`migrate down-to <previous version>`, and the pipeline goes red. The old colour
stays up after a good deploy: it is the instant `rollback:<app>:1`.

The first blue-green deploy of budget migrates the old in-place stack: traffic
stays on the old `backend`/`frontend` services until blue is healthy, Caddy is
recreated once with the directory mount (a second or two), then the old
containers are removed.
