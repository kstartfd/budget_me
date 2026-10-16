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

## Production

GitLab CI tests, builds and deploys Learning to the shared VPS
(`https://learning.budgetme.xyz`) when `apps/learning/` or `infra/learning/`
changes. How it is wired — Compose project, edge Caddy, secrets on the box,
backups, rebuilding — is in [`infra/learning/README.md`](../../infra/learning/README.md).

`apps/learning/scripts/copy-learning-data.sh` copies Learning accounts and content
out of the old shared budget database into an empty, migrated Learning database
(user IDs, password hashes, topics, entries, topic-scoped extension tokens and
capture jobs). It was used once, on 2026-10-16, with the row counts verified.

## Local checks

```sh
cd apps/learning/backend && go test ./...
cd ../frontend && pnpm install --frozen-lockfile && pnpm typecheck
```
