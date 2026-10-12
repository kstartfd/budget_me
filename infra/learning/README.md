# Learning deployment draft

The current GitLab pipeline tests and builds Learning images but does not deploy
them. The current budget VPS and its Caddy configuration stay unchanged.

At the server migration:

1. Provision the new VPS and `docker network create apps_edge`.
2. Give budget and learning separate Compose projects, databases, secrets and
   image tags. Keep only the shared Caddy on ports 80/443.
3. Change budget Compose to join `apps_edge`, put the shared Caddyfile in place,
   and start the budget stack. Start `infra/learning/compose/prod.yml` with its
   own `.env` and run its migration job.
4. Migrate learning data with `apps/learning/scripts/copy-learning-data.sh`,
   validate counts and app logins, then point the learning subdomain at the VPS.
5. Add a learning deploy job with its own GitLab resource group, health gate and
   rollback, plus a separate encrypted off-server database backup.

`infra/edge/Caddyfile.shared.example` is a routing template. It is intentionally
not mounted by the budget deployment today.

The old budget database still holds the existing Learning tables. The numbered
Learning migrations remain in the budget backend until the data has been copied
and checked on the new server; removing or reversing them now would risk losing
the only production copy. Budget no longer serves Learning API routes.
