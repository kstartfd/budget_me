# budget_me — CI/CD (GitLab → VPS)

This is the **primary** deploy path. `git push` to `main` builds the backend and
frontend Docker images on the GitLab runner, pushes them to the GitLab Container
Registry, and the **deploy** job SSHes into the VPS to pull + restart the stack.
Every green `main` pipeline auto-deploys (no manual gate). The VPS never builds
images. The pipeline is defined in [`.gitlab-ci.yml`](../.gitlab-ci.yml); the
manual fallback lives in [`DEPLOY.md`](DEPLOY.md).

```
push main ─▶ build:backend ─┐            (Docker-in-Docker on the runner)
            build:frontend ─┴─▶ push to registry.gitlab.com/kstartfd/budget_me
                                    │
                                    ▼
                              deploy (auto): ssh deploy@78.140.240.60
                                render ~/budget_me/.env  ·  docker login
                                compose pull → up -d → migrate → /healthz gate
```

- Project: `gitlab.com/kstartfd/budget_me`
- Registry: `registry.gitlab.com/kstartfd/budget_me`
  (`…/backend:<sha>`, `…/frontend:<sha>`, plus a moving `:latest`)
- Server: `deploy@78.140.240.60` (Ubuntu 26.04, Docker + Compose v2, 2 GB RAM,
  Moscow). Root bootstrap → `deploy` user; see §2.
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
ssh-keyscan 78.140.240.60
```
Copy the whole output (all lines).

### 1d. Add the CI/CD variables
Settings → CI/CD → **Variables**. For each: set **Protected** (only protected
refs — `main`/tags — run these jobs) and **Masked** where marked. `SSH_PRIVATE_KEY`
is a multi-line key, so it **cannot be masked** — set its **Type = File** instead.

| Variable                | Value                                          | Type     | Protected | Masked |
|-------------------------|------------------------------------------------|----------|-----------|--------|
| `SSH_PRIVATE_KEY`       | contents of `~/.ssh/budget_me_deploy`          | **File** | yes       | —      |
| `SSH_HOST`              | `78.140.240.60`                                | Variable | yes       | no     |
| `SSH_KNOWN_HOSTS`       | output of `ssh-keyscan 78.140.240.60`          | Variable | yes       | no     |
| `DEPLOY_TOKEN_USER`     | deploy-token username (from §1b)               | Variable | yes       | no     |
| `DEPLOY_TOKEN_PASSWORD` | deploy-token secret (from §1b)                 | Variable | yes       | **yes**|
| `DOMAIN`                | `budgetme.xyz`                                 | Variable | yes       | no     |
| `POSTGRES_PASSWORD`     | `openssl rand -hex 24`                         | Variable | yes       | **yes**|
| `AUTH_SECRET`           | `openssl rand -hex 32`                          | Variable | yes       | **yes**|
| `TELEGRAM_BOT_TOKEN`    | @BotFather token (may start empty)             | Variable | yes       | **yes**|

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
# deploy user + docker access
useradd -m -s /bin/bash deploy
usermod -aG docker deploy

# authorize the CI deploy key (contents of ~/.ssh/budget_me_deploy.pub)
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
echo 'ssh-ed25519 AAAA…  gitlab-ci' > /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys && chown -R deploy:deploy /home/deploy/.ssh

# directories the pipeline writes into (compose + infra/caddy/Caddyfile + scripts)
sudo -u deploy mkdir -p /home/deploy/budget_me/infra/caddy /home/deploy/budget_me/scripts

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
3. The deploy job renders `~/budget_me/.env`, `docker login`s, then
   `compose pull && up -d && migrate` behind a `/healthz` gate with rollback.
4. Open `https://budgetme.xyz`. First hit may lag a few seconds while Caddy
   fetches the TLS cert — follow it with
   `ssh deploy@78.140.240.60 'cd ~/budget_me && docker compose -f prod.yml logs -f caddy'`.

### Where the deploy job puts files (and how compose reads them)
The `./infra/caddy/Caddyfile` bind-mount in `prod.yml` resolves against the
compose file's own directory, so the layout on the server is:

```
~/budget_me/
├── prod.yml                     # scp of infra/compose/prod.yml
├── .env                         # rendered from CI variables, chmod 600
└── infra/caddy/Caddyfile        # scp of infra/caddy/Caddyfile  → mounted read-only
```

The job runs `cd ~/budget_me && docker compose --env-file .env -f prod.yml …`, so
the mount resolves to `~/budget_me/infra/caddy/Caddyfile` — exactly where the job
placed it.

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
push main → test + build:* go green → deploy runs automatically
```

Every green `main` pipeline auto-deploys (`deploy` has no `when: manual`).
Each deploy pins `BACKEND_IMAGE` / `FRONTEND_IMAGE` to the new commit SHA in
`.env`, re-runs migrations (idempotent), and gates on `/healthz` — a failing
step rolls the previous images back and reds the pipeline.

## 5. Rollback

Re-run the deploy job of an older green pipeline (its `.env` pins that older
SHA), or SSH in and point `BACKEND_IMAGE` / `FRONTEND_IMAGE` in `~/budget_me/.env`
at a previous `:<sha>` then `docker compose --env-file .env -f prod.yml up -d`.
