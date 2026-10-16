#!/usr/bin/env bash
# Off-box database backup: pg_dump → gzip → gpg(AES-256) → Telegram.
#
# The one rule a backup has to obey is that it does not share a disk with the
# thing it backs up: this box holds the only copy of every user's finances, so a
# dead volume is a dead app. The dump therefore leaves the machine. It goes to
# Telegram (sendDocument) through the bot the app already owns — no bucket, no
# credentials, no monthly bill, and the file lands in a chat the owner reads.
#
# It is encrypted BEFORE it leaves: the dump carries transaction history and
# password hashes, and Telegram is not a safe place for either in the clear.
# Symmetric gpg with BACKUP_PASSPHRASE — the passphrase lives in the GitLab CI
# variables and in the owner's password manager, NOT on this box, or a stolen
# box would hand over both the data and the key to read it.
#
# Installed by the deploy at ~/budget_me/scripts/backup.sh and run by cron (see
# infra/budget/cron/budget_me-backup). Restore instructions: infra/budget/DEPLOY.md.
#
#   ./scripts/backup.sh            dump, encrypt, ship to Telegram, prune local
#   ./scripts/backup.sh --local    dump + encrypt only (the deploy runs this
#                                  right before migrations — a bad migration is
#                                  then one gunzip away from being undone)
#
# BACKUP_LOCAL_ONLY=1 in the env file forces --local for every run — for a box
# that cannot reach api.telegram.org (e.g. hosted in RU). The local copies then
# ARE the backup, so the keep window still applies and they are still encrypted.
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-prod.yml}"
BACKUP_DIR="${BACKUP_DIR:-backups}"
# Local copies are a convenience for a fast restore, not the backup — keep a
# week of them and let the off-box copies be the ones that matter.
KEEP_LOCAL="${KEEP_LOCAL:-7}"
# One script serves every app on the box: the defaults are budget_me's, another
# app overrides them from its crontab line (infra/learning/cron/install-crons.sh).
APP_NAME="${APP_NAME:-budget_me}"
DB_USER="${DB_USER:-budget_me}"
DB_NAME="${DB_NAME:-budget_me}"

LOCAL_ONLY=""
[ "${1:-}" = "--local" ] && LOCAL_ONLY=1

# An explicitly exported value beats the file. The deploy renders a line per CI
# variable, so an unset BACKUP_PASSPHRASE lands in .env as an EMPTY assignment —
# sourcing that would blank a value passed on the command line, and a manual run
# or a restore drill would have to edit the live .env to get anywhere.
env_pass="${BACKUP_PASSPHRASE:-}"
env_chat="${BACKUP_CHAT_ID:-}"
env_token="${TELEGRAM_BOT_TOKEN:-}"

case "$ENV_FILE" in
  /*) ;;                      # absolute
  *) ENV_FILE="./$ENV_FILE" ;; # `.` without a slash would search PATH
esac

set -a
# shellcheck disable=SC1090 # path is a variable by design (dev vs prod env file)
. "$ENV_FILE"
set +a

[ -n "$env_pass" ] && BACKUP_PASSPHRASE="$env_pass"
[ -n "$env_chat" ] && BACKUP_CHAT_ID="$env_chat"
[ -n "$env_token" ] && TELEGRAM_BOT_TOKEN="$env_token"

# A box that can't reach Telegram opts out of the upload for good via the env
# file; the flag on the command line still wins on top of that.
case "${BACKUP_LOCAL_ONLY:-}" in ""|0|false|no) ;; *) LOCAL_ONLY=1 ;; esac

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

# Shout into the same chat the dumps go to. Best-effort: a backup must not fail
# because the notification failed.
notify() {
  [ -n "${TELEGRAM_BOT_TOKEN:-}" ] && [ -n "${BACKUP_CHAT_ID:-}" ] || return 0

  curl -sS --max-time 30 \
    "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
    -d chat_id="$BACKUP_CHAT_ID" \
    -d text="$1" >/dev/null 2>&1 || true
}

fail() {
  echo "backup: $1" >&2
  notify "🔴 $APP_NAME: бэкап НЕ сделан — $1"

  exit 1
}

[ -n "${BACKUP_PASSPHRASE:-}" ] || fail "BACKUP_PASSPHRASE не задан (см. infra/budget/DEPLOY.md)"

if [ -z "$LOCAL_ONLY" ]; then
  [ -n "${TELEGRAM_BOT_TOKEN:-}" ] || fail "TELEGRAM_BOT_TOKEN не задан"
  [ -n "${BACKUP_CHAT_ID:-}" ] || fail "BACKUP_CHAT_ID не задан (см. infra/budget/DEPLOY.md)"
fi

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

stamp="$(date -u +%Y-%m-%d_%H%M)"
dump="$BACKUP_DIR/$APP_NAME-$stamp.sql.gz.gpg"

# -T: cron gives no tty. </dev/null: `compose exec` ATTACHES STDIN even with -T,
# and this script is called from the deploy, whose whole body is piped into a
# remote `bash -s` over ssh — so pg_dump quietly ate the rest of the deploy
# script (migrations, up -d, the health gate) and the pipeline went green having
# changed nothing. The dump reads no input; say so.
#
# The plaintext never touches the disk: it streams straight through gzip and gpg.
compose exec -T postgres pg_dump -U "$DB_USER" --clean --if-exists "$DB_NAME" </dev/null \
  | gzip -9 \
  | gpg --batch --yes --quiet \
        --symmetric --cipher-algo AES256 \
        --passphrase-fd 3 --output "$dump" 3<<<"$BACKUP_PASSPHRASE" \
  || fail "pg_dump/шифрование упало"

chmod 600 "$dump"

size="$(du -h "$dump" | cut -f1)"
[ -s "$dump" ] || fail "дамп пустой"

prune_local() {
  # Newest first, drop everything past the keep window. `|| true`: pipefail
  # would let a hiccup in the cleanup mark a backup that already ran as failed.
  ls -1t "$BACKUP_DIR"/"$APP_NAME"-*.sql.gz.gpg 2>/dev/null \
    | tail -n +$((KEEP_LOCAL + 1)) \
    | xargs -r rm -f \
    || true
}

if [ -n "$LOCAL_ONLY" ]; then
  prune_local
  echo "backup: $dump ($size), локально ($KEEP_LOCAL шт., без Telegram)"

  exit 0
fi

# The upload IS the backup — anything short of a confirmed ok:true from Telegram
# means this night has no off-box copy, and that has to be loud.
response="$(
  curl -sS --max-time 300 \
    "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendDocument" \
    -F chat_id="$BACKUP_CHAT_ID" \
    -F document="@$dump" \
    -F caption="$APP_NAME · дамп БД · $stamp UTC · $size · gpg AES-256"
)" || fail "отправка в Telegram упала (сеть)"

case "$response" in
  *'"ok":true'*) ;;
  *) fail "Telegram отклонил дамп: $(printf '%s' "$response" | head -c 200)" ;;
esac

prune_local

echo "backup: $dump ($size) отправлен в Telegram, локально храним $KEEP_LOCAL шт."
