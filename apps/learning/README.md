# Learnings

Separate Telegram Mini App with its own Go backend, Next.js frontend, PostgreSQL
schema, session secret and Telegram bot. The design copied from the budget app
is retained.

## Local development

From the repository root:

```sh
make learning.up
```

The command creates a local, ignored `apps/learning/.env` with a generated
`AUTH_SECRET` if needed, builds the current backend and frontend, applies
migrations, and checks the frontend API proxy. Set `LEARNING_API_PORT` to change
the host API port (default `8082`); the frontend uses `3001`. Run
`make learning.down` to stop the stack. Re-run `make learning.up` after code changes.

Frontend: `http://localhost:3001`. The new bot's production Mini App URL will
be `https://<learning-domain>` after the VPS move.

## Production

GitHub Actions tests, builds and deploys Learning to the shared VPS
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
