# budget_me

Personal budgeting app delivered as a **Telegram Mini App**.

- **Backend** — Go + Echo + Postgres (pgx + sqlc), hexagonal layout, goose migrations.
- **Frontend** — Next.js + shadcn/ui + TanStack Query, running inside Telegram's webview.

The backend is a **clean skeleton**: the folder architecture, project launch
and SQL/migration pipeline are in place; domain features are built on top from
here. The frontend already ships the Mini App screens (overview, transactions,
add, budgets, settings) running on a **mock data layer** (seeded, persisted to
localStorage) behind the same TanStack Query hooks the real API will use —
wiring the backend later means swapping fetchers in `features/budget/api.ts`
only.

## Layout

```
backend/    Go modular monolith (api → application → domain ← adapter)
frontend/   Next.js Telegram Mini App
infra/      Docker Compose dev/test stacks + backend Dockerfile
Makefile    Task runner (see `make help`)
```

## Getting started

Backend stack (Postgres + Redis + backend with hot reload) runs in Docker:

```bash
make up            # start postgres, redis, backend (http on :8080)
make migrate.up    # apply migrations
make help          # all targets
```

Frontend (Next dev server on :3000):

```bash
cd frontend && pnpm install
pnpm dev
```

To develop against a real Telegram client, expose the dev server over HTTPS
(e.g. a tunnel) and set it as your bot's Mini App URL. In a plain browser the app
runs too — it just can't read Telegram user data (`window.Telegram.WebApp` is
absent), so it shows the out-of-Telegram fallback.

## Backend workflow

```bash
make sqlc          # regenerate queries from backend/queries/*.sql
make build         # go build ./...
make vet           # go vet ./...
make test          # go test ./...
```

SQL lives only in `backend/queries/*.sql` (sqlc) and `backend/migrations/*.sql`
(goose). See `backend/CODING_STYLE.md` and `.claude/CLAUDE.md` for conventions.
