# Infrastructure

`budget/` owns the budget stack and its automatic GitLab CI rollout; its Caddy is
the single edge (ports 80/443) for the whole box. `learning/` owns the learning
stack — its own Compose project, database, bot and secrets — and its
`deploy:learning` job. `edge/` holds the Caddyfile that routes both domains.
