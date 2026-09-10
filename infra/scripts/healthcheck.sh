#!/usr/bin/env bash
# On-box health check, run by cron every 5 minutes. Watches the things that kill
# a 1 GB VPS — disk, memory, load, a container that quietly died, a database the
# app can no longer reach — and the thing that silently makes an outage
# unrecoverable: backups that stopped happening.
#
# It sends through the bot the app already owns, like backup.sh. Deliberately not
# Prometheus/Grafana: on one core and 961 MB they would compete for the very
# memory they are supposed to report on.
#
# THIS CANNOT BE THE ONLY MONITOR. A check that runs on the box cannot fire when
# the box is what died (OOM, full disk, kernel panic) — pair it with an external
# uptime check on https://<DOMAIN>/healthz (see infra/DEPLOY.md).
#
# Antispam is the whole design: 288 runs a day means a naive script would send a
# message every 5 minutes for as long as a disk stays full, and the operator
# would learn to ignore the channel. So it alerts on a CHANGE of state, says so
# when things recover, and only nags every REMIND_HOURS while a problem persists.
#
#   ./scripts/healthcheck.sh          check, alert on state changes
#   DRY_RUN=1 ./scripts/healthcheck.sh   print what it WOULD send, touch nothing
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-prod.yml}"
STATE_FILE="${STATE_FILE:-.healthcheck-state}"

# An exported value beats the file (see backup.sh: the deploy renders empty
# assignments for unset CI variables, and sourcing those would blank them).
env_chat="${ALERT_CHAT_ID:-}"
env_token="${TELEGRAM_BOT_TOKEN:-}"

case "$ENV_FILE" in
  /*) ;;
  *) ENV_FILE="./$ENV_FILE" ;;
esac

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

[ -n "$env_chat" ] && ALERT_CHAT_ID="$env_chat"
[ -n "$env_token" ] && TELEGRAM_BOT_TOKEN="$env_token"

# One chat serves both the dumps and the alerts until the noise justifies two.
CHAT_ID="${ALERT_CHAT_ID:-${BACKUP_CHAT_ID:-}}"

# Thresholds. Sized for THIS box (1 core, 961 MB, 9.8 GB): load 2.0 on one core
# already means a queue, and 15% of 961 MB is the point where Postgres starts to
# feel it.
DISK_MAX_PCT="${DISK_MAX_PCT:-85}"
MEM_MIN_AVAIL_PCT="${MEM_MIN_AVAIL_PCT:-15}"
LOAD_MAX="${LOAD_MAX:-2.0}"
# 26h, not 24: the nightly dump runs at 03:40 and a check at 03:30 must not call
# a backup that is about to happen "missing".
BACKUP_MAX_AGE_H="${BACKUP_MAX_AGE_H:-26}"
REMIND_HOURS="${REMIND_HOURS:-6}"
SERVICES="${SERVICES:-caddy backend frontend postgres}"

DRY_RUN="${DRY_RUN:-}"

now="$(date +%s)"

# State per check: status, when it went bad, and when we last said so. The last
# one is what keeps a persistent problem from re-alerting on every run.
declare -A state_status state_since state_notified

if [ -f "$STATE_FILE" ]; then
  while IFS='|' read -r key status since notified; do
    [ -n "${key:-}" ] || continue

    state_status["$key"]="$status"
    state_since["$key"]="$since"
    state_notified["$key"]="${notified:-$since}"
  done < "$STATE_FILE"
fi

declare -A new_status new_since new_notified

send() {
  if [ -n "$DRY_RUN" ]; then
    echo "[dry-run] $1"

    return 0
  fi

  [ -n "${TELEGRAM_BOT_TOKEN:-}" ] && [ -n "$CHAT_ID" ] || return 0

  # The response is checked, not discarded: a wrong chat id answers 400 and the
  # alerts would otherwise vanish into it — a monitor that believes it is
  # reporting while nobody hears it is worse than no monitor.
  local response
  response="$(
    curl -sS --max-time 30 \
      "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
      -d chat_id="$CHAT_ID" \
      -d text="$1" 2>&1 || echo '{"ok":false,"description":"curl failed"}'
  )"

  case "$response" in
    *'"ok":true'*) ;;
    *) echo "healthcheck: АЛЕРТ НЕ ДОСТАВЛЕН: $(printf '%s' "$response" | head -c 160)" >&2 ;;
  esac
}

# report <key> <ok|bad> <message>
#
# The state machine: alert when a check FLIPS, say "recovered" when it flips
# back, and while it stays bad remind once every REMIND_HOURS — never every run.
report() {
  local key="$1" status="$2" message="$3"
  local was="${state_status[$key]:-ok}"
  local since="${state_since[$key]:-$now}"
  local notified="${state_notified[$key]:-0}"

  new_status["$key"]="$status"

  if [ "$status" = "bad" ]; then
    if [ "$was" != "bad" ]; then
      new_since["$key"]="$now"
      new_notified["$key"]="$now"

      send "🔴 budget_me: $message"

      return 0
    fi

    new_since["$key"]="$since"
    new_notified["$key"]="$notified"

    # Nag from the LAST message, not from a whole-hour boundary: `elapsed % 6`
    # was true for every run inside that hour — twelve identical reminders, and
    # a channel the owner learns to ignore.
    if [ $(( now - notified )) -ge $(( REMIND_HOURS * 3600 )) ]; then
      new_notified["$key"]="$now"

      send "🔴 budget_me ($(( (now - since) / 3600 ))ч без изменений): $message"
    fi

    return 0
  fi

  new_since["$key"]="$now"
  new_notified["$key"]="$now"

  if [ "$was" = "bad" ]; then
    send "✅ budget_me: $message"
  fi
}

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

# ── disk ─────────────────────────────────────────────────────────────────────
disk_pct="$(df -P / | awk 'NR==2 {gsub("%","",$5); print $5}')"
disk_free="$(df -Ph / | awk 'NR==2 {print $4}')"

if [ "$disk_pct" -ge "$DISK_MAX_PCT" ]; then
  report disk bad "диск занят ${disk_pct}% (порог ${DISK_MAX_PCT}%), свободно ${disk_free}"
else
  report disk ok "диск в норме — ${disk_pct}%, свободно ${disk_free}"
fi

# ── memory ───────────────────────────────────────────────────────────────────
mem_avail_pct="$(free -m | awk '/Mem:/ {printf "%d", $7*100/$2}')"
mem_avail_mb="$(free -m | awk '/Mem:/ {print $7}')"

if [ "$mem_avail_pct" -le "$MEM_MIN_AVAIL_PCT" ]; then
  report memory bad "памяти доступно ${mem_avail_pct}% (${mem_avail_mb} МБ), порог ${MEM_MIN_AVAIL_PCT}%"
else
  report memory ok "память в норме — доступно ${mem_avail_pct}% (${mem_avail_mb} МБ)"
fi

# ── load ─────────────────────────────────────────────────────────────────────
load5="$(awk '{print $2}' /proc/loadavg)"

if awk -v l="$load5" -v m="$LOAD_MAX" 'BEGIN {exit !(l > m)}'; then
  report load bad "нагрузка ${load5} за 5 мин (порог ${LOAD_MAX}, ядер $(nproc))"
else
  report load ok "нагрузка в норме — ${load5}"
fi

# ── containers ───────────────────────────────────────────────────────────────
# `|| true` on every pipeline whose head may legitimately fail: pipefail would
# otherwise turn "docker is unhappy" into a health check that dies silently —
# the monitor going quiet exactly when it has something to say.
running="$(compose ps --status running --format '{{.Service}}' 2>/dev/null | tr '\n' ' ' || true)"
missing=""

for svc in $SERVICES; do
  case " $running " in
    *" $svc "*) ;;
    *) missing="$missing $svc" ;;
  esac
done

if [ -n "$missing" ]; then
  report containers bad "контейнеры не работают:${missing}"
else
  report containers ok "все контейнеры на месте ($SERVICES)"
fi

# ── the app, end to end ──────────────────────────────────────────────────────
# Through Caddy and TLS, not the internal network: this is the path a user takes,
# and /healthz answers 503 when the DB is gone — "process up" is not the question.
#
# curl's `-w '%{http_code}'` already prints 000 when it never got a response
# (DNS, TCP, TLS, timeout), so a `|| echo 000` here appended a SECOND 000 and the
# alert read "отвечает 000000". `|| true` only swallows the non-zero exit that
# set -e would abort on; the `:=` covers the paranoid case of an empty capture.
health_code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 \
  "https://${DOMAIN:-budgetme.xyz}/healthz" || true)"
: "${health_code:=000}"

if [ "$health_code" != "200" ]; then
  report health bad "/healthz отвечает ${health_code} (ждём 200)"
else
  report health ok "/healthz снова 200"
fi

# ── backups actually happening ───────────────────────────────────────────────
# The failure nobody notices until it is too late: the cron died, or was never
# configured, and there has been no dump for weeks.
#
# The directory may not exist yet (no dump ever taken) — and with pipefail a
# failing `find` took the whole check down with it, silently, before it could say
# so. That bug hid the very alert this block exists to raise.
backup_dir="${BACKUP_DIR:-backups}"
newest_backup=""

if [ -d "$backup_dir" ]; then
  newest_backup="$(
    find "$backup_dir" -name 'budget_me-*.sql.gz.gpg' -type f -printf '%T@\n' 2>/dev/null \
      | sort -n | tail -1 | cut -d. -f1 || true
  )"
fi

if [ -z "$newest_backup" ]; then
  report backup bad "бэкапов нет вообще (BACKUP_PASSPHRASE/BACKUP_CHAT_ID заданы? см. infra/DEPLOY.md)"
else
  age_h=$(( (now - newest_backup) / 3600 ))

  if [ "$age_h" -ge "$BACKUP_MAX_AGE_H" ]; then
    report backup bad "последнему бэкапу ${age_h}ч (порог ${BACKUP_MAX_AGE_H}ч) — ночной дамп не сделался"
  else
    report backup ok "бэкап свежий — ${age_h}ч назад"
  fi
fi

# ── persist ──────────────────────────────────────────────────────────────────
if [ -z "$DRY_RUN" ]; then
  : > "$STATE_FILE"

  for key in "${!new_status[@]}"; do
    echo "${key}|${new_status[$key]}|${new_since[$key]}|${new_notified[$key]}" >> "$STATE_FILE"
  done

  chmod 600 "$STATE_FILE"
fi

bad_count=0

for key in "${!new_status[@]}"; do
  [ "${new_status[$key]}" = "bad" ] && bad_count=$((bad_count + 1))
done

echo "healthcheck: диск ${disk_pct}% | память ${mem_avail_pct}% | load ${load5} | /healthz ${health_code} | проблем: ${bad_count}"
