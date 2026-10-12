# Infrastructure

`budget/` owns the currently deployed budget stack and its automatic GitLab CI
rollout. `learning/` contains a prepared Compose stack, Dockerfiles and a local
development stack; there is no production deploy job yet. `edge/` contains the
active budget-only Caddyfile and a shared-domain template for the server move.
