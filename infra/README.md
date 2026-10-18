# Infrastructure

`budget/` owns the host stack of the VPS (Caddy, Postgres) and the budget app,
deployed blue-green by GitLab CI (`budget/CI_CD.md`). `learning/` holds the
learning app's local stack and its production compose, which joins the host
stack with its own database and colours. `edge/` holds the Caddyfile (both
sites) and the first-boot `*.active.caddy` files that `budget/scripts/bluegreen.sh`
owns on the server. `edge/Caddyfile.shared.example` is superseded by
`edge/Caddyfile`.
