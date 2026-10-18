#!/usr/bin/env bash
# Blue-green deploy and rollback for the apps on this VPS (budget, learning).
#
# Each app runs its backend + frontend twice, as a "blue" and a "green" colour;
# Caddy routes the app's domain to ONE of them through
# infra/edge/<app>.active.caddy. Caddy and Postgres are shared by both apps
# (host stack = prod.yml); every app has its own database, role, releases log
# and lock, so deploying or rolling back one never touches the other.
#
#   ./scripts/bluegreen.sh <app> deploy <backend-image> <frontend-image>
#   ./scripts/bluegreen.sh <app> rollback <steps>   # 1 = previous release, 2 = the one before
#   ./scripts/bluegreen.sh <app> status
#
# deploy:   backup → migrate UP with the new image → start the idle colour →
#           health → switch Caddy → watch the public URL → record the release.
#           A failure at any step before the release is recorded puts traffic
#           back on the old colour and the schema back where it was.
# rollback: start the target release in the idle colour (or find it still
#           running there) → health → switch Caddy → backup → migrate DOWN to
#           the schema that release was deployed with, using the NEWER image
#           (only it carries the Down SQL of its own migrations).
#           ROLLBACK_KEEP_SCHEMA=1 skips the down: use it when the migrations in
#           between are additive and the data written since must stay.
#
# The order is mirrored on purpose: the code serving traffic never sees a
# schema OLDER than the one it was built for. Down migrations delete what was
# written into the dropped columns/tables since the deploy — that is the price
# of rolling the schema back, and why a backup runs right before.
#
# releases-<app>.log gets one line per successful deploy:
#   <utc time> <colour> <backend image> <frontend image> <schema version>
# Rollbacks read it and never write it, so `rollback 2` always means "two
# successful deploys before the last one" and repeating it changes nothing.
set -euo pipefail

cd "$(dirname "$0")/.."

# The deploy pipes its script into a remote `bash -s` over ssh, and compose
# run/exec/up attach stdin by default — they would eat the rest of the deploy.
# Nothing here reads input.
exec </dev/null

fail() {
  echo "bluegreen: $1" >&2

  exit 1
}

usage() {
  fail "usage: bluegreen.sh <budget|learning> deploy <backend-image> <frontend-image> | rollback <steps> | status"
}

APP="${1:-}"
shift || true

# The host stack: Caddy + Postgres live in budget's compose project.
HOST_ENV_FILE=".env"
HOST_COMPOSE_FILE="prod.yml"
NETWORK="budget_me_default"
EDGE_DIR="infra/edge"

case "$APP" in
  budget)
    ENV_FILE=".env"
    COMPOSE_FILE="prod.yml"
    PROJECT="budget_me"
    PREFIX=""
    MIGRATE="migrate"
    DB_NAME="budget_me"
    # Before blue-green this app ran as plain backend/frontend services.
    LEGACY_BACKEND="backend"
    LEGACY_FRONTEND="frontend"
    ;;
  learning)
    ENV_FILE=".env.learning"
    COMPOSE_FILE="learning.yml"
    PROJECT="learning_app"
    PREFIX="learning-"
    MIGRATE="learning-migrate"
    DB_NAME="learning_app"
    LEGACY_BACKEND=""
    LEGACY_FRONTEND=""
    ;;
  *)
    usage
    ;;
esac

RELEASES="releases-$APP.log"
ROLLBACKS="rollbacks-$APP.log"
ACTIVE_CADDY="$EDGE_DIR/$APP.active.caddy"
HEALTH_TRIES="${HEALTH_TRIES:-20}"
# After the switch the public URL is polled VERIFY_TRIES times, 3s apart;
# VERIFY_MAX_FAILS failures in a row switch traffic back.
VERIFY_TRIES="${VERIFY_TRIES:-20}"
VERIFY_MAX_FAILS="${VERIFY_MAX_FAILS:-3}"

# Two runs for the same app must never interleave. CI already serialises them
# with a per-app resource_group; this covers a manual run on the box.
exec 9>".bluegreen-$APP.lock"
flock -n 9 || fail "$APP: уже идёт другой deploy/rollback"

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

host_compose() {
  docker compose --env-file "$HOST_ENV_FILE" -f "$HOST_COMPOSE_FILE" "$@"
}

env_get() {
  grep -E "^$1=" "$ENV_FILE" | tail -1 | cut -d= -f2- || true
}

env_set() {
  sed -i "/^$1=/d" "$ENV_FILE"
  printf '%s=%s\n' "$1" "$2" >> "$ENV_FILE"
}

other() {
  if [ "$1" = blue ]; then echo green; else echo blue; fi
}

upper() {
  printf '%s' "$1" | tr '[:lower:]' '[:upper:]'
}

# Empty on a box that has not run a blue-green deploy of this app yet.
active_color() {
  env_get ACTIVE_COLOR
}

# migrate_with <image> <args…> — the migrate service with an explicit image.
# A shell variable beats the --env-file value in compose interpolation.
migrate_with() {
  local image="$1"

  shift
  BACKEND_IMAGE="$image" compose --profile tools run --rm -T "$MIGRATE" "$@"
}

schema_version() {
  migrate_with "$1" version 2>/dev/null | sed -n 's/^version: *//p' | tail -1
}

backup_db() {
  BACKUP_DB="$DB_NAME" ./scripts/backup.sh --local \
    || echo "bluegreen: локальный бэкап $DB_NAME не сделан — продолжаю" >&2
}

# The learning database and role live in the shared Postgres. Idempotent: the
# password is re-applied on every deploy, so rotating the CI variable is enough.
ensure_learning_db() {
  local password

  password="$(env_get POSTGRES_PASSWORD)"
  [ -n "$password" ] || fail "learning: POSTGRES_PASSWORD пуст в $ENV_FILE"

  host_compose up -d postgres

  for _ in $(seq 1 20); do
    host_compose exec -T postgres pg_isready -U budget_me -d budget_me >/dev/null 2>&1 && break
    sleep 2
  done

  host_compose exec -T postgres psql -U budget_me -d postgres -v ON_ERROR_STOP=1 -v pw="$password" <<'SQL' >/dev/null
SELECT 'CREATE ROLE learning_app LOGIN'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'learning_app')\gexec
ALTER ROLE learning_app WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD :'pw';
SELECT 'CREATE DATABASE learning_app OWNER learning_app'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'learning_app')\gexec
-- Each app reaches only its own database.
REVOKE CONNECT ON DATABASE budget_me FROM PUBLIC;
REVOKE CONNECT ON DATABASE learning_app FROM PUBLIC;
GRANT CONNECT ON DATABASE learning_app TO learning_app;
SQL
}

# caddy_routes <backend host> <frontend host> — the body of an *.active.caddy.
#
# lb_try_duration: a request that lands while an upstream restarts is retried
# for up to 5s instead of failing with 502. fail_duration: an upstream that just
# failed is skipped for 10s (passive health check).
caddy_routes() {
  local backend="$1" frontend="$2"

  cat <<EOF
# Rewritten by scripts/bluegreen.sh at $(date -u +%FT%TZ) — do not edit by hand.
#
# The backend serves /api/v1/*, so the FULL path is forwarded (\`handle\` keeps
# the /api prefix; \`handle_path\` would strip it). /healthz is exposed for an
# external uptime monitor of whichever colour is live.
handle /api/* {
	reverse_proxy $backend:8080 {
		lb_try_duration 5s
		fail_duration 10s
	}
}

handle /healthz {
	reverse_proxy $backend:8080
}

handle {
	reverse_proxy $frontend:3000 {
		lb_try_duration 5s
		fail_duration 10s
	}
}
EOF
}

# Caddy refuses to start when an imported file is missing, so both apps' files
# must exist before any switch. A first-boot budget file points at the
# pre-blue-green services (still serving), learning's at blue (502 until its
# first deploy).
ensure_edge_files() {
  [ -f "$EDGE_DIR/budget.active.caddy" ] || caddy_routes backend frontend > "$EDGE_DIR/budget.active.caddy"
  [ -f "$EDGE_DIR/learning.active.caddy" ] || caddy_routes learning-backend-blue learning-frontend-blue > "$EDGE_DIR/learning.active.caddy"
}

health() {
  local color="$1"

  for _ in $(seq 1 "$HEALTH_TRIES"); do
    # A throwaway curl container on the shared network checks both halves of
    # the colour by service name. /healthz answers 503 when the DB is
    # unreachable, so this is more than "the process started".
    if docker run --rm --network "$NETWORK" --entrypoint sh curlimages/curl:latest -c \
         "curl -fsS -o /dev/null http://${PREFIX}backend-$color:8080/healthz && curl -fsS -o /dev/null http://${PREFIX}frontend-$color:3000/" \
         >/dev/null 2>&1; then
      echo "bluegreen: $APP/$color здоров"

      return 0
    fi

    sleep 3
  done

  echo "bluegreen: $APP/$color не прошёл health-check за $((HEALTH_TRIES * 3))с" >&2

  return 1
}

# verify_public — the path a user takes: Caddy, TLS, the live colour. Run from
# the shared network with --connect-to, so it does not depend on the box
# resolving its own domain; -k because a brand-new domain may still be waiting
# for its certificate (TLS itself is healthcheck.sh's business).
verify_public() {
  local domain fails=0

  domain="$(env_get DOMAIN)"
  [ -n "$domain" ] || { echo "bluegreen: DOMAIN не задан — пропускаю проверку снаружи" >&2; return 0; }

  for _ in $(seq 1 "$VERIFY_TRIES"); do
    if docker run --rm --network "$NETWORK" curlimages/curl:latest \
         -fsSk -o /dev/null --max-time 5 --connect-to "$domain:443:caddy:443" \
         "https://$domain/healthz" >/dev/null 2>&1; then
      fails=0
    else
      fails=$((fails + 1))
      echo "bluegreen: https://$domain/healthz не ответил ($fails подряд)" >&2
      [ "$fails" -lt "$VERIFY_MAX_FAILS" ] || return 1
    fi

    sleep 3
  done

  echo "bluegreen: https://$domain отвечает стабильно"
}

# start_color <colour> <backend image> <frontend image>
#
# Called as `if ! start_color …`, where bash switches `set -e` off for the whole
# function body — hence an explicit `|| return 1` on every step.
start_color() {
  local color="$1" backend="$2" frontend="$3" up got

  up="$(upper "$color")"
  env_set "${up}_BACKEND_IMAGE" "$backend" || return 1
  env_set "${up}_FRONTEND_IMAGE" "$frontend" || return 1

  compose pull "${PREFIX}backend-$color" "${PREFIX}frontend-$color" || return 1
  compose up -d "${PREFIX}backend-$color" "${PREFIX}frontend-$color" || return 1

  # The container must actually BE the image we asked for: a deploy that
  # silently keeps the old image and reports success is worse than a failure.
  got="$(docker inspect "${PROJECT}-${PREFIX}backend-${color}-1" --format '{{.Config.Image}}')" || return 1
  [ "$got" = "$backend" ] || { echo "bluegreen: ${PREFIX}backend-$color на $got, ждали $backend" >&2; return 1; }

  health "$color"
}

stop_color() {
  compose stop "${PREFIX}backend-$1" "${PREFIX}frontend-$1" || true
}

# point_caddy <backend host> <frontend host> — rewrite this app's routes and
# reload. Rolls the file back if Caddy refuses the config, so a failed switch
# leaves traffic where it was. No `set -e` in here either (called under `if !`).
point_caddy() {
  local backup="$ACTIVE_CADDY.prev"

  cp -f "$ACTIVE_CADDY" "$backup" || return 1
  # mv, not >: Caddy never reads a half-written file.
  caddy_routes "$1" "$2" > "$ACTIVE_CADDY.tmp" || return 1
  mv -f "$ACTIVE_CADDY.tmp" "$ACTIVE_CADDY" || return 1

  # `up -d` recreates Caddy only when its container spec changed (the first
  # blue-green deploy moves it to the infra/edge directory mount); otherwise it
  # is a no-op and the reload does the zero-downtime switch.
  if ! host_compose up -d caddy \
     || ! host_compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile; then
    mv -f "$backup" "$ACTIVE_CADDY"
    host_compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile || true

    echo "bluegreen: Caddy не принял переключение $APP на $1" >&2

    return 1
  fi

  rm -f "$backup"
}

switch_to() {
  point_caddy "${PREFIX}backend-$1" "${PREFIX}frontend-$1" || return 1
  env_set ACTIVE_COLOR "$1" || return 1

  echo "bluegreen: трафик $APP переключён на $1"
}

# switch_back <previous colour or empty> — undo a switch whose release did not
# hold up. Empty = the pre-blue-green services.
switch_back() {
  if [ -n "$1" ]; then
    switch_to "$1"
  elif [ -n "$LEGACY_BACKEND" ]; then
    point_caddy "$LEGACY_BACKEND" "$LEGACY_FRONTEND" && env_set ACTIVE_COLOR ""
  fi
}

# undo_schema <image> <version> — the failure path of a deploy that already
# migrated up.
undo_schema() {
  local image="$1" version="$2"

  [ -n "$version" ] || return 0

  if [ "$(schema_version "$image")" != "$version" ]; then
    migrate_with "$image" down-to "$version" \
      || echo "bluegreen: СХЕМА НЕ ОТКАЧЕНА до $version — нужен ручной разбор" >&2
  fi
}

cmd_deploy() {
  local backend="$1" frontend="$2" current idle live_backend schema_before schema

  ensure_edge_files

  if [ "$APP" = learning ]; then
    ensure_learning_db
  fi

  host_compose up -d postgres

  current="$(active_color)"
  # First blue-green deploy of this app: nothing is a colour yet, start with blue.
  if [ -n "$current" ]; then idle="$(other "$current")"; else idle=blue; fi

  live_backend="$(env_get BACKEND_IMAGE)"
  schema_before="$(schema_version "$backend")"

  echo "bluegreen: $APP активен ${current:-legacy}, выкатываю $backend в $idle (схема ${schema_before:-?})"

  # Migrations BEFORE the switch: the old colour keeps serving on an expanded
  # schema, the new colour starts on the schema it was built for.
  backup_db
  if ! migrate_with "$backend" up; then
    undo_schema "$backend" "$schema_before"
    fail "$APP: миграции не прошли — трафик остался на ${current:-legacy}"
  fi

  if ! start_color "$idle" "$backend" "$frontend"; then
    stop_color "$idle"
    undo_schema "$backend" "$schema_before"
    fail "$APP: релиз не поднялся в $idle — трафик остался на ${current:-legacy}"
  fi

  if ! switch_to "$idle"; then
    stop_color "$idle"
    undo_schema "$backend" "$schema_before"
    fail "$APP: переключение на $idle не удалось — трафик остался на ${current:-legacy}"
  fi

  # Nothing to go back to on an app's very first deploy (and its brand-new
  # domain may still be waiting for a certificate): skip the watch then.
  if [ -z "$current" ] && [ -z "$LEGACY_BACKEND" ]; then
    echo "bluegreen: первый деплой $APP — проверку снаружи пропускаю"
  elif ! verify_public; then
    echo "bluegreen: $APP/$idle не держит трафик — возвращаю ${current:-legacy}" >&2
    switch_back "$current" || fail "$APP: ВОЗВРАТ НЕ УДАЛСЯ — трафик на $idle, разбирать вручную"
    stop_color "$idle"
    undo_schema "$backend" "$schema_before"
    fail "$APP: релиз откатан автоматически после переключения"
  fi

  # The tools-profile services (migrate/admin/worker and the cron jobs) run
  # the live release.
  env_set BACKEND_IMAGE "$backend"
  env_set FRONTEND_IMAGE "$frontend"

  # Leftovers of the pre-blue-green stack: once a colour serves traffic they
  # are dead weight. No-op on every later deploy.
  if [ -n "$LEGACY_BACKEND" ]; then
    docker rm -f "${PROJECT}-${LEGACY_BACKEND}-1" "${PROJECT}-${LEGACY_FRONTEND}-1" >/dev/null 2>&1 || true
  fi

  schema="$(schema_version "$backend")"
  printf '%s %s %s %s %s\n' "$(date -u +%FT%TZ)" "$idle" "$backend" "$frontend" "${schema:-?}" >> "$RELEASES"

  echo "bluegreen: $APP на $idle, схема ${schema:-?}; предыдущий релиз (${live_backend:-—}) остаётся в ${current:-—}"
}

cmd_rollback() {
  local steps="$1" total line backend frontend schema current idle up live_backend current_schema

  case "$steps" in
    ''|*[!0-9]*|0) fail "rollback: число шагов должно быть >= 1, получено '$steps'" ;;
  esac

  total=0
  [ -f "$RELEASES" ] && total="$(wc -l < "$RELEASES")"
  [ "$total" -gt "$steps" ] || fail "$APP: в $RELEASES $total релиз(ов) — на $steps назад откатиться нельзя"

  # The release `steps` deploys before the last one.
  line="$(tail -n "$((steps + 1))" "$RELEASES" | head -n 1)"
  read -r _ _ backend frontend schema <<<"$line"

  current="$(active_color)"
  [ -n "$current" ] || fail "$APP: ACTIVE_COLOR не задан — нужен хотя бы один blue-green deploy"

  live_backend="$(env_get BACKEND_IMAGE)"

  if [ "$live_backend" = "$backend" ] && [ "$(env_get FRONTEND_IMAGE)" = "$frontend" ]; then
    echo "bluegreen: $APP уже на $backend — делать нечего"

    return 0
  fi

  idle="$(other "$current")"
  up="$(upper "$idle")"

  echo "bluegreen: $APP откат на $steps назад → $backend (в $idle)"

  # One step back usually finds the target still running in the idle colour:
  # then this is a health check and a switch — no pull, no restart.
  if [ "$(env_get "${up}_BACKEND_IMAGE")" = "$backend" ] \
     && [ "$(env_get "${up}_FRONTEND_IMAGE")" = "$frontend" ]; then
    compose up -d "${PREFIX}backend-$idle" "${PREFIX}frontend-$idle"
    health "$idle" || fail "$APP: $idle не здоров — трафик остался на $current"
  elif ! start_color "$idle" "$backend" "$frontend"; then
    stop_color "$idle"
    fail "$APP: откат не поднялся в $idle — трафик остался на $current"
  fi

  switch_to "$idle" || fail "$APP: переключение на $idle не удалось — трафик остался на $current"

  if ! verify_public; then
    switch_back "$current" || true
    fail "$APP: откатанный релиз не держит трафик — вернул $current"
  fi

  env_set BACKEND_IMAGE "$backend"
  env_set FRONTEND_IMAGE "$frontend"

  # Schema AFTER the switch: the old code now serving tolerates the newer
  # schema for these seconds (expand/contract), never the other way round.
  current_schema="$(schema_version "$live_backend")"
  if [ "$schema" = "?" ] || [ -z "$current_schema" ]; then
    echo "bluegreen: версия схемы неизвестна — схему не трогаю" >&2
  elif [ "$schema" = "$current_schema" ]; then
    echo "bluegreen: схема уже $schema"
  elif [ "${ROLLBACK_KEEP_SCHEMA:-}" = 1 ]; then
    echo "bluegreen: ROLLBACK_KEEP_SCHEMA=1 — схема остаётся $current_schema (релиз записан с $schema)"
  else
    backup_db
    # The image that was live knows every migration up to $current_schema,
    # Down SQL included; the target image does not.
    migrate_with "$live_backend" down-to "$schema" \
      || fail "$APP: код откатан на $backend, но схема НЕ откатилась с $current_schema до $schema"
    echo "bluegreen: схема $APP откатана $current_schema → $schema"
  fi

  printf '%s rollback-%s %s %s %s\n' "$(date -u +%FT%TZ)" "$steps" "$idle" "$backend" "$frontend" >> "$ROLLBACKS"

  echo "bluegreen: $APP откатан, живой цвет $idle"
}

cmd_status() {
  echo "$APP active: $(active_color)"
  echo "blue:   $(env_get BLUE_BACKEND_IMAGE) $(env_get BLUE_FRONTEND_IMAGE)"
  echo "green:  $(env_get GREEN_BACKEND_IMAGE) $(env_get GREEN_FRONTEND_IMAGE)"
  echo "releases (последние 5):"
  tail -n 5 "$RELEASES" 2>/dev/null || echo "  нет"
}

case "${1:-}" in
  deploy)
    [ $# -eq 3 ] || usage
    cmd_deploy "$2" "$3"
    ;;
  rollback)
    [ $# -eq 2 ] || usage
    cmd_rollback "$2"
    ;;
  status)
    cmd_status
    ;;
  *)
    usage
    ;;
esac
