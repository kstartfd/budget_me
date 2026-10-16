#!/usr/bin/env bash
# Ensures the deploy user's crontab carries the jobs the learning app owns. Run by
# the deploy:learning job over ssh on every green main — same idea and same rules
# as infra/budget/cron/install-crons.sh (it edits the SAME crontab, so the two
# deploys share one resource_group and never write it at once).
#
#   backup   03:45 daily — pg_dump → gpg (→ Telegram, unless BACKUP_LOCAL_ONLY is set)
#            It is infra/budget/scripts/backup.sh run against the learning DB.
#
# Idempotent: a job is matched by its command, so re-running changes nothing and
# an operator who retimed a job keeps their edit.
set -euo pipefail

# A fresh box has no crontab yet: `crontab -l` then exits 1. Capture it ONCE with
# `|| true` — inside a `( … )` under `set -e` the subshell would abort before the
# new line is appended.
current="$(crontab -l 2>/dev/null || true)"

ensure() {
  local match="$1" comment="$2" line="$3"

  if printf '%s\n' "$current" | grep -qF "$match"; then
    echo "cron: $match уже установлен"

    return 0
  fi

  current="$(printf '%s\n' "$current" "$comment" "$line")"

  echo "cron: $match установлен"
}

ensure 'learning && APP_NAME=learning' \
  '# learning: ночной бэкап БД' \
  '45 3 * * * cd ~/learning && APP_NAME=learning DB_USER=learning_app DB_NAME=learning_app ./scripts/backup.sh >> ~/learning/backup.log 2>&1'

# One write, at the end — a partial crontab is never installed.
printf '%s\n' "$current" | crontab -
