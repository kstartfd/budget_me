# Learning — production deployment

Runs on the same box as budget (`deploy@45.133.19.54`), served at
`https://learning.budgetme.xyz` by its own Telegram bot (`@nexyrium_bot`).

## Shape

- **Own Compose project** `learning_app` in `~/learning` on the box
  ([`compose/prod.yml`](compose/prod.yml)): its own Postgres + volume, backend,
  frontend, migrate job. Nothing is shared with budget except the box, the
  registry token and the edge below.
- **Edge:** the Caddy in the *budget* project is the only thing on ports 80/443. It
  joins the external `apps_edge` network and routes `LEARNING_DOMAIN` to the
  aliases `learning-api` / `learning-web` that this stack publishes there
  ([`../edge/Caddyfile`](../edge/Caddyfile)). `LEARNING_DOMAIN` comes from
  `~/budget_me/.env.local` (it falls back to `learning.invalid` if missing, so a
  forgotten value can never break budget's Caddy config).
- **CI:** `test:learning-*` → `build:learning-*` → `deploy:learning`, run when
  `apps/learning/`, `infra/learning/` or `.gitlab-ci.yml` change. The deploy has
  the same safety as budget's: env staged as `.env.new`, local backup and
  migrations before the switch, the containers must be the shipped images,
  `/healthz` gate, rollback on any error. It shares `resource_group: production`
  with budget's deploy (shared crontab and shared Caddy) and runs after it when
  both are in one pipeline.
- **Bot:** learning needs **its own bot** — two pollers on one token fight over
  `getUpdates` (409) and would break budget's bot. In @BotFather the Mini App URL
  of the learning bot is `https://learning.budgetme.xyz`.

## Secrets live on the box

`~/learning/.env.local` (chmod 600, never in git) holds `POSTGRES_PASSWORD`,
`AUTH_SECRET`, `DOMAIN`, `TELEGRAM_BOT_TOKEN`, `BACKUP_PASSPHRASE` and the backup
destination. The deploy merges it into the rendered `.env`
([`../budget/scripts/merge-env.sh`](../budget/scripts/merge-env.sh)). They are
deliberately **not** CI variables: the CI variables named `POSTGRES_PASSWORD`,
`AUTH_SECRET`, `DOMAIN` and `TELEGRAM_BOT_TOKEN` belong to budget and would point
learning at budget's database and bot.

`AUTH_SECRET` signs the sessions **and** keys the encrypted copy of every extension
token that lets the app show a token again. Keep it stable and carry it along with
any database restore: after a change everyone is signed out and issued tokens keep
working but can no longer be shown (`GET /tokens/<id>/secret` answers 500).

## Backups

Cron `03:45` (installed by the deploy, [`cron/install-crons.sh`](cron/install-crons.sh))
runs budget's [`backup.sh`](../budget/scripts/backup.sh) with `APP_NAME=learning
DB_USER=learning_app DB_NAME=learning_app` from `~/learning`: `pg_dump` → gzip →
gpg AES-256 → `~/learning/backups/learning-*.sql.gz.gpg` (7 kept).

While `BACKUP_LOCAL_ONLY=1` is in `.env.local` the dumps stay on the box. To ship
them to Telegram remove that line and set `BACKUP_CHAT_ID` to a chat or channel the
**learning** bot can post to (the dump is sent by `TELEGRAM_BOT_TOKEN`, i.e. the
learning bot). Restore:

```bash
gpg --batch --passphrase "$BACKUP_PASSPHRASE" -d learning-<stamp>.sql.gz.gpg | gunzip \
  | docker compose --env-file .env -f prod.yml exec -T postgres psql -U learning_app -d learning_app
```

## Rebuilding the box

1. `docker network create apps_edge` (once), then give budget its
   `LEARNING_DOMAIN` in `~/budget_me/.env.local`.
2. Create `~/learning/.env.local` with the keys above (`openssl rand -hex 24` /
   `-hex 32` for the two secrets).
3. Let the pipeline deploy — it creates the database, volume and runs migrations.
4. Restore the latest backup (above). Only for an empty database coming from the
   old shared one, use [`apps/learning/scripts/copy-learning-data.sh`](../../apps/learning/scripts/copy-learning-data.sh).

## History

On 2026-10-16 the existing learning data (1 user, 2 topics, 5 entries, 6 tokens,
6 capture jobs) was copied out of the budget database with
`copy-learning-data.sh`, counts verified equal. The budget database still holds its
old `learning_*` tables and migrations `0021`–`0023`; they are intentionally left in
place (budget no longer serves any Learning route) until the owner decides to drop them.
