#!/usr/bin/env bash
# Ensures the deploy user's crontab carries the jobs this repo owns. Run by the
# deploy job over ssh on every green main, so a rebuilt box gets its backups and
# its health checks back without anyone remembering to install a cron by hand.
#
# Idempotent: each job is matched by its script path, so re-running changes
# nothing and an operator who retimed a job keeps their edit.
#
#   backup      03:40 daily — pg_dump → gpg → Telegram (infra/scripts/backup.sh)
#   healthcheck every 5 min — disk/memory/load/containers/healthz/backup age
#
# The worker job (03:17) predates this script and is left exactly as it is.
set -euo pipefail

ensure() {
  local match="$1" comment="$2" line="$3"

  if crontab -l 2>/dev/null | grep -q "$match"; then
    echo "cron: $match уже установлен"

    return 0
  fi

  (crontab -l 2>/dev/null; echo "$comment"; echo "$line") | crontab -

  echo "cron: $match установлен"
}

ensure 'scripts/backup.sh' \
  '# budget_me: ночной бэкап БД off-VPS' \
  '40 3 * * * cd ~/budget_me && ./scripts/backup.sh >> ~/budget_me/backup.log 2>&1'

ensure 'scripts/healthcheck.sh' \
  '# budget_me: health-чек (диск/память/нагрузка/контейнеры/healthz/свежесть бэкапа)' \
  '*/5 * * * * cd ~/budget_me && ./scripts/healthcheck.sh >> ~/budget_me/healthcheck.log 2>&1'
