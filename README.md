# Budget Me workspace

One GitLab repository with two separate Telegram Mini Apps.

| Application | Source | Runtime |
| --- | --- | --- |
| Budget and lists | `apps/budget/` | Go API, Next.js, PostgreSQL, existing Telegram bot |
| Learnings | `apps/learning/` | Go API, Next.js, separate PostgreSQL and bot |

Both apps share one account (Telegram / email login, profile, deletion):
`libs/account` (see its README) over a shared `account_app` database.

Shared deployment configuration lives in `infra/`. Both apps run on one VPS
behind one Caddy and one Postgres (separate databases), each deployed
**blue-green** by GitLab CI: every `main` pipeline builds both apps and offers
four manual buttons — `deploy:budget`, `deploy:learning`, `rollback:budget`,
`rollback:learning` (see `infra/budget/CI_CD.md`).

## Local work

```sh
make up             # budget Postgres + Go API
make migrate.up
make front          # budget Next.js
```

For Learnings local setup, see `apps/learning/README.md`.

## Deployment

- `infra/budget/`: current production Compose, Dockerfiles, cron and scripts.
- `infra/learning/`: local and production Compose (production joins the budget host stack).
- `infra/edge/Caddyfile`: both sites; each imports its `*.active.caddy` (live colour).
- `infra/budget/scripts/bluegreen.sh`: deploy / rollback / status per app.

Deployment details are in `infra/budget/CI_CD.md`; the learning cutover is in
`infra/learning/README.md`.
