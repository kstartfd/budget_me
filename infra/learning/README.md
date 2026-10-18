# Learning deployment

Learning runs on the same VPS as budget, at `learning.<domain>`
(`LEARNING_DOMAIN`, default `learning.budgetme.xyz`):

- `infra/learning/compose/prod.yml` is shipped as `~/budget_me/learning.yml`
  (compose project `learning_app`). It has no Caddy and no Postgres of its own:
  it joins `budget_me_default`, and the shared Postgres holds its own database
  and role `learning_app` (created idempotently by `bluegreen.sh learning deploy`;
  `PUBLIC` loses `CONNECT` on both databases, so neither app reaches the other's).
- Blue-green like budget: `learning-backend-blue/green`,
  `learning-frontend-blue/green`, routed by `infra/edge/learning.active.caddy`.
- CI: `deploy:learning`, `rollback:learning:1`, `rollback:learning:2` exist only
  with `LEARNING_DEPLOY_ENABLED=true` and the `LEARNING_*` variables
  (infra/budget/CI_CD.md §1d).
- Nightly backup of `learning_app` runs from the same cron as budget's.

Cutover, once:

1. Point the `learning.<domain>` A record at the VPS.
2. Add the `LEARNING_*` CI variables and `LEARNING_DEPLOY_ENABLED=true`; push
   or re-run the pipeline. The first deploy creates the database, migrates it
   and starts blue (no switch-back on a first deploy).
3. Copy the data out of the budget database with
   `apps/learning/scripts/copy-learning-data.sh` (stop writes to the old
   learning API first), validate counts and logins.
4. Point the learning bot's Mini App URL at `https://learning.<domain>`.

The old budget database still holds the existing Learning tables. The numbered
Learning migrations remain in the budget backend until the data has been copied
and checked; removing or reversing them now would risk losing the only
production copy. Budget no longer serves Learning API routes.
