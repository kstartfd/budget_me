# budget_me — CI/CD (GitLab → VPS)

This is the **primary** deploy path. `git push` to `main` builds the backend and
frontend Docker images on the GitLab runner, pushes them to the GitLab Container
Registry, and a manual **deploy** job SSHes into the VPS to pull + restart the
stack. The VPS never builds images (it has ~1 GB RAM). The pipeline is defined in
[`.gitlab-ci.yml`](../.gitlab-ci.yml); the manual fallback lives in
[`DEPLOY.md`](DEPLOY.md).

```
push main ─▶ build:backend ─┐            (Docker-in-Docker on the runner)
            build:frontend ─┴─▶ push to registry.gitlab.com/kstartfd/budget_me
                                    │
                                    ▼
                              deploy (manual): ssh deploy@45.140.204.139
                                render ~/budget_me/.env  ·  docker login
                                compose pull → up -d → migrate
```

- Project: `gitlab.com/kstartfd/budget_me`
- Registry: `registry.gitlab.com/kstartfd/budget_me`
  (`…/backend:<sha>`, `…/frontend:<sha>`, plus a moving `:latest`)
- Server: `deploy@45.140.204.139` (Ubuntu 24.04, Docker + Compose v2)
- Domain: `budgetme.xyz`

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
ssh-keyscan 45.140.204.139
```
Copy the whole output (all lines).

### 1d. Add the CI/CD variables
Settings → CI/CD → **Variables**. For each: set **Protected** (only protected
refs — `main`/tags — run these jobs) and **Masked** where marked. `SSH_PRIVATE_KEY`
is a multi-line key, so it **cannot be masked** — set its **Type = File** instead.

| Variable                | Value                                          | Type     | Protected | Masked |
|-------------------------|------------------------------------------------|----------|-----------|--------|
| `SSH_PRIVATE_KEY`       | contents of `~/.ssh/budget_me_deploy`          | **File** | yes       | —      |
| `SSH_HOST`              | `45.140.204.139`                               | Variable | yes       | no     |
| `SSH_KNOWN_HOSTS`       | output of `ssh-keyscan 45.140.204.139`         | Variable | yes       | no     |
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

## 2. One-time server bootstrap (as `deploy`)

Nothing to build and no repo to clone — the deploy job ships the compose file,
Caddyfile and a rendered `.env`. Only prepare the host:

```bash
# authorize the CI deploy key (paste budget_me_deploy.pub)
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo 'ssh-ed25519 AAAA…  budget_me-deploy' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys

# directory the pipeline writes into (compose + infra/caddy/Caddyfile)
mkdir -p ~/budget_me/infra/caddy

# firewall — only SSH + HTTP/HTTPS (Postgres is never published)
sudo ufw allow 22 && sudo ufw allow 80 && sudo ufw allow 443 && sudo ufw enable
```

`deploy` must be able to run Docker without sudo (`sudo usermod -aG docker deploy`,
then re-login). **Swap is not needed** — image builds happen in CI, not here.

**DNS:** create an A record `budgetme.xyz → 45.140.204.139` and wait for it to
resolve (`dig +short budgetme.xyz`) before the first deploy, or Caddy's first
Let's Encrypt challenge will fail and retry.

---

## 3. First release

1. Push the branch to `main`:
   ```bash
   git push origin fresh-budget_me:main
   ```
2. Watch the pipeline (CI/CD → Pipelines). `build:backend` + `build:frontend`
   should go green (they build and push `:<sha>` and `:latest`).
3. Click **Play** on the manual **deploy** job. It renders `~/budget_me/.env`,
   `docker login`s, then `compose pull && up -d && migrate`.
4. Open `https://budgetme.xyz`. First hit may lag a few seconds while Caddy
   fetches the TLS cert — follow it with
   `ssh deploy@45.140.204.139 'cd ~/budget_me && docker compose -f prod.yml logs -f caddy'`.

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
push main → build:* go green → click Play on deploy
```

`deploy` is `when: manual` so production only changes on a deliberate click.
Each deploy pins `BACKEND_IMAGE` / `FRONTEND_IMAGE` to the new commit SHA in
`.env` and re-runs migrations (idempotent). **To auto-deploy** every green `main`
build, delete the `when: manual` line from the `deploy` job in `.gitlab-ci.yml`.

## 5. Rollback

Re-run the deploy job of an older green pipeline (its `.env` pins that older
SHA), or SSH in and point `BACKEND_IMAGE` / `FRONTEND_IMAGE` in `~/budget_me/.env`
at a previous `:<sha>` then `docker compose --env-file .env -f prod.yml up -d`.
