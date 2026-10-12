# budget_me — production deploy runbook (manual fallback)

> **The primary deploy path is CI/CD — see [`CI_CD.md`](CI_CD.md).** `git push`
> to `main` builds the images in GitLab, pushes them to the registry, and a
> manual **deploy** job pulls + restarts the stack on the VPS. This runbook is
> the hand-driven fallback (bring the stack up yourself from a checkout). Either
> way the images are **pulled from the registry** — nothing is built on the VPS.

Target: one small VPS (Ubuntu 26.04, Docker + Compose v2, 2 GB RAM, ~24 GB
disk — `deploy@78.140.240.60`). Everything runs from `infra/compose/prod.yml`:

- **caddy** — the only internet-facing service (ports 80/443), automatic HTTPS
  via Let's Encrypt, path-routes `/api/*` → backend and everything else → frontend.
- **frontend** — Next.js standalone server, internal `:3000`.
- **backend** — Go/Echo API, internal `:8080`, migrations embedded.
- **postgres** — internal only, no published ports.
- **migrate** — one-off runner (profile `tools`).

All commands below are run **from the repo root** and pass the env file
explicitly so there is no working-directory ambiguity.

---

## 0. Swap — not needed

Images are built in CI, not on the server, so there is no `next build` to OOM
the box; **no swap is required**. (Add some only if you ever build locally.)

## 1. Point DNS at the server

Create an **A record** for your domain (e.g. `budget.example.com`) → the VPS
public IP. Wait for it to resolve (`dig +short budget.example.com`) before
starting Caddy, otherwise the ACME challenge fails and Caddy retries.

## 2. Get the code onto the server

The `make prod.*` targets drive compose from a checkout (they only need
`prod.yml`, the Caddyfile and the Makefile — the images come from the registry):

```bash
git clone <repo-url> budget_me && cd budget_me
# later updates: git pull
```

## 3. Log the server into the registry

Images are private. Log in once with a `read_registry` **deploy token** (create
it in GitLab: Settings → Repository → Deploy tokens — see [`CI_CD.md` §1b](CI_CD.md)):

```bash
docker login registry.gitlab.com -u <deploy-token-user> -p <deploy-token-secret>
```

## 4. Create the production .env

The env file lives at `infra/compose/.env` (all `make prod.*` targets point
`--env-file` there). Copy the template and fill it in:

```bash
cp .env.prod.example infra/compose/.env
# generate secrets:
openssl rand -hex 24   # -> POSTGRES_PASSWORD
openssl rand -hex 32   # -> AUTH_SECRET
nano infra/compose/.env
```

Set `DOMAIN`, `POSTGRES_PASSWORD`, `AUTH_SECRET`, `TELEGRAM_BOT_TOKEN` (may stay
empty for the first bring-up), and the image tags **`BACKEND_IMAGE` /
`FRONTEND_IMAGE`** — pin a real `:<sha>` from a green pipeline (or `:latest`).
**Never commit this file** — the repo `.gitignore` already ignores every `.env`.

## 5. Pull and start

```bash
make prod.up          # docker compose pull, then up -d
# or explicitly:
docker compose --env-file infra/compose/.env --project-directory . \
    -f infra/compose/prod.yml pull
docker compose --env-file infra/compose/.env --project-directory . \
    -f infra/compose/prod.yml up -d
```

`--project-directory .` makes the `./infra/caddy/Caddyfile` mount resolve from
the repo root. First run pulls both images and Caddy fetches the TLS cert; watch
it with `docker compose -f infra/compose/prod.yml logs -f caddy`.

## 6. Run database migrations

```bash
make prod.migrate
# or:
docker compose --env-file infra/compose/.env --project-directory . \
    -f infra/compose/prod.yml --profile tools run --rm migrate
```

This runs `/migrate up` against Postgres (waits for it to be healthy first).
Re-run it any time new migrations are added.

## 7. Category presets (seeded automatically)

The 6 shared presets (`user_id IS NULL`, identified by `slug`) are seeded by
migration `0004_seed_categories.sql`, which the migrate step (§6) applies
automatically and idempotently (`ON CONFLICT (slug) DO NOTHING`). Nothing to do
by hand — the UI localizes each preset by its `slug`, so `name` stays NULL.

## 8. Register the Telegram Mini App

In @BotFather: `/newapp` (or Bot Settings → Mini App) and set the Web App URL to
`https://<DOMAIN>`. Put the resulting bot token into `infra/compose/.env`
(`TELEGRAM_BOT_TOKEN`) and restart the backend:

```bash
docker compose --env-file infra/compose/.env --project-directory . \
    -f infra/compose/prod.yml up -d backend
```

## 9. Firewall

```bash
sudo ufw allow 22
sudo ufw allow 80
sudo ufw allow 443
sudo ufw enable
```

Postgres is never published, so nothing else needs opening.

## Backups (off-VPS, nightly)

This box holds the only live copy of every user's finances, so the dump does not
stay on it. `scripts/backup.sh` (from [`infra/scripts/backup.sh`](scripts/backup.sh),
installed by the deploy) does: `pg_dump` → `gzip` → `gpg` AES-256 → **Telegram**
`sendDocument` through the bot the app already owns. No bucket, no bill, and the
file lands in a chat you read. Cron runs it at **03:40** daily (installed by the
deploy, see [`infra/cron/install-backup-cron.sh`](cron/install-backup-cron.sh));
the deploy also takes a local dump right before running migrations.

The dump carries transaction history and password hashes, so it is encrypted
**before** it leaves the box. The passphrase lives in the GitLab CI variables and
in your password manager — deliberately NOT on the server, or a stolen box would
hand over the data and the key to read it together.

### The two values it needs

| Value | What |
|---|---|
| `BACKUP_PASSPHRASE` | gpg symmetric passphrase. Generate: `openssl rand -base64 32`. **Without it the dumps are noise** — keep a copy somewhere you will still have if the server is gone. |
| `BACKUP_CHAT_ID` | Where the dumps go: your Telegram user id (the bot can DM you — you already started it) or a private channel with the bot as admin. |

Two places they can live, and the deploy handles both:

1. **GitLab CI/CD → Variables** (`BACKUP_PASSPHRASE` masked + protected). The
   passphrase then never touches the server, so a stolen box cannot hand over the
   dumps and the key to read them in one go.
2. **`~/budget_me/.env.local` on the server** (chmod 600, never in git). The
   deploy renders `.env` from the CI variables and then merges this file into
   whatever CI left empty (`scripts/merge-env.sh`), so a redeploy no longer
   blanks it. Simpler, and the trade it makes is honest: an attacker who owns the
   box already owns the live database — the encryption still protects the copies
   sitting in Telegram. This is what budget_me actually runs on.

CI wins whenever it has a value, so moving a secret from the box into GitLab
later is just setting it there.

With the value in neither place the nightly job fails loudly into
`~/budget_me/backup.log` (and, once a chat id exists, announces the failure in
that chat) — a silent "no backups" is the one outcome worth engineering against.

### Restore

```bash
# 1. get the .gpg file out of the Telegram chat onto the box (or your laptop)
# 2. decrypt + restore. --clean --if-exists is baked into the dump, so this
#    replaces the current contents rather than colliding with them:
gpg --batch --passphrase "$BACKUP_PASSPHRASE" -d budget_me-<stamp>.sql.gz.gpg \
  | gunzip \
  | docker compose --env-file .env -f prod.yml exec -T postgres \
      psql -U budget_me -d budget_me

# restore into a scratch DB first if you only want to look:
docker compose --env-file .env -f prod.yml exec -T postgres \
  psql -U budget_me -d postgres -c 'CREATE DATABASE budget_me_restore;'
```

Run a restore drill occasionally — a backup nobody has restored is a hypothesis,
not a backup.

---

## Alerts

Three layers, because each one is blind to what the others see.

**On the box** — [`infra/scripts/healthcheck.sh`](scripts/healthcheck.sh), cron
every 5 minutes (installed by the deploy). Watches disk (>85%), available memory
(<15%), 5-min load (>2.0 on one core), the four containers, `/healthz` through
Caddy and TLS, and **how old the newest backup is** (>26h) — the cron dying
quietly is how a backup strategy actually fails. Thresholds are sized for this
box (1 core, 961 MB, 9.8 GB) and overridable per run:
`DISK_MAX_PCT=90 ./scripts/healthcheck.sh`.

It alerts on a **change of state**, says "✅ …" when a problem clears, and only
nags every 6h while one persists. 288 runs a day would otherwise train you to
mute the channel. Dry run (prints, sends nothing, writes nothing):

```bash
DRY_RUN=1 ./scripts/healthcheck.sh
```

**In the app** — `internal/alert`: every 5xx passes through the single
`httpx.ErrorHandler`, so alerting needed no per-handler plumbing. The first
failure on a route pages immediately; the rest of a storm are folded into one
"ещё N× …" summary per 10-minute window, per route. Routes are reported as
patterns (`/api/v1/transactions/:id`), never as URLs with somebody's id in them,
and no request bodies are ever sent — the bodies here are somebody's money.

**Outside the box — the one you must set up yourself.** An alert that runs on
the VPS cannot fire when the VPS is what died (OOM, full disk, kernel panic), so
an external check is not optional:

1. [UptimeRobot](https://uptimerobot.com) (free) → **Add New Monitor**
2. Type `HTTPS`, URL `https://budgetme.xyz/healthz`, interval 5 min
3. Alert contact → Telegram → point it at the same chat

`/healthz` answers `503` when the DB is unreachable, so this catches "the process
is up but the app is dead", not just a down box. (Not done as a GitLab schedule
on purpose: a 15-minute cron is ~2 900 pipelines a month and would eat the free
CI minutes; UptimeRobot exists for exactly this.)

### The alert chat

| Variable | What |
|---|---|
| `ALERT_CHAT_ID` | Where alerts go. Optional — falls back to `BACKUP_CHAT_ID`, so one chat serves both the dumps and the alerts until the noise justifies two. |

With neither set, alerting is **off**: the reporter is a no-op and the cron sends
nothing. That is the right behaviour for a dev box, and the deploy prints a
warning so it is never a silent surprise in prod.

---

## Updating a running deployment

Normally you don't — push to `main` and click **deploy** (see [`CI_CD.md`](CI_CD.md)).
By hand: bump `BACKEND_IMAGE` / `FRONTEND_IMAGE` in `infra/compose/.env` to the
new `:<sha>` (or keep `:latest`), then:

```bash
cd budget_me
make prod.up          # pull + up -d
make prod.migrate     # only if new migrations landed
```

## Diagnostics

```bash
# service status / logs
docker compose -f infra/compose/prod.yml ps
docker compose -f infra/compose/prod.yml logs -f backend       # or caddy / frontend
docker compose -f infra/compose/prod.yml exec backend /migrate status

# disk & cleanup (each release pulls a fresh image; watch the ~5 GB budget)
df -h
docker system prune -af          # remove unused/old images
docker image prune -f

# backend health (run from caddy — the distroless backend image has no shell)
docker compose -f infra/compose/prod.yml exec caddy wget -qO- http://backend:8080/healthz || true
```

If Caddy cannot get a certificate: confirm the DNS A record resolves to this
host, ports 80/443 are open, and check `docker compose logs caddy`.
