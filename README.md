# Budget Me workspace

One GitLab repository with two separate Telegram Mini Apps.

| Application | Source | Runtime |
| --- | --- | --- |
| Budget and lists | `apps/budget/` | Go API, Next.js, PostgreSQL, existing Telegram bot |
| Learnings | `apps/learning/` | Go API, Next.js, separate PostgreSQL and bot |

Shared deployment configuration lives in `infra/`. The budget application keeps
its current automatic GitLab deployment to the existing VPS. Learnings is
checked and built in CI, but has **no production deploy job** until the VPS
migration.

## Local work

```sh
make up             # budget Postgres + Go API
make migrate.up
make front          # budget Next.js
```

For Learnings local setup, see `apps/learning/README.md`.

## Deployment

- `infra/budget/`: current production Compose, Dockerfiles, cron and scripts.
- `infra/learning/`: prepared local and production Compose; production is not active.
- `infra/edge/Caddyfile`: current budget-only Caddy configuration.
- `infra/edge/Caddyfile.shared.example`: routing template for the future shared VPS.

Budget deployment details are in `infra/budget/CI_CD.md`; the future learning
cutover is outlined in `infra/learning/README.md`.
