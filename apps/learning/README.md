# Learnings

Separate Telegram Mini App with its own Go backend, Next.js frontend, PostgreSQL
schema, session secret and Telegram bot. The design copied from the budget app
is retained.

## Local development

From the repository root, copy `apps/learning/.env.example` to
`apps/learning/.env` and set a local `AUTH_SECRET` and bot URL. Then:

```sh
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml up -d --build postgres
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml --profile tools run --rm migrate
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml up -d --build backend frontend
```

Frontend: `http://localhost:3001`. The new bot's production Mini App URL will
be `https://<learning-domain>` after the VPS move.

## Production preparation

GitLab CI tests and builds `learning-backend` and `learning-frontend` images
when `apps/learning/` or `infra/learning/` changes. No learning deploy job is
enabled yet. `infra/learning/compose/prod.yml` is a draft for the future shared
VPS. It expects the `apps_edge` Docker network and a single Caddy configured
with `infra/edge/Caddyfile.shared.example`. Do not start it on the current VPS.

After the new VPS is available, configure the new bot, domain, secrets and
registry pull credentials, then copy the learning data before switching users
to the new bot. The data transfer script is
`apps/learning/scripts/copy-learning-data.sh`; it requires both database URLs
and a migrated, empty target database. It preserves user IDs, password hashes,
topics, entries, topic-scoped extension tokens and capture jobs. Stop writes to
the old learning API during the copy and verify the row counts afterward.

## Local checks

```sh
cd apps/learning/backend && go test ./...
cd ../frontend && pnpm install --frozen-lockfile && pnpm typecheck
```
