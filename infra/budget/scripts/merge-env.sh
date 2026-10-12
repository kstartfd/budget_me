#!/usr/bin/env bash
# Merge box-local secrets into the .env the deploy just rendered.
#
# The deploy writes one line per CI variable, so anything not set in CI lands as
# an EMPTY assignment and overwrites whatever was there before. That makes the CI
# variables the only possible home for a secret — fine for a team, heavy for a
# pet project where the operator would rather keep a value on the server.
#
# So: ~/budget_me/.env.local (chmod 600, never in git, never touched by the
# deploy) fills in the keys CI left empty. CI still WINS when it has a value, so
# moving a secret into GitLab later is just setting it there — nothing to undo
# here.
#
#   ./scripts/merge-env.sh .env.new .env.local
set -euo pipefail

target="${1:-.env.new}"
local_file="${2:-.env.local}"

[ -f "$target" ] || { echo "merge-env: нет $target" >&2; exit 1; }
[ -f "$local_file" ] || { echo "merge-env: $local_file нет — пропускаю"; exit 0; }

merged=0

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ''|'#'*) continue ;;
    *'='*) ;;
    *) continue ;;
  esac

  key="${line%%=*}"
  current="$(grep -E "^${key}=" "$target" | tail -1 | cut -d= -f2- || true)"

  # A key CI actually set wins; an empty (or absent) one is filled from the box.
  if [ -n "$current" ]; then
    continue
  fi

  sed -i "/^${key}=/d" "$target"
  printf '%s\n' "$line" >> "$target"

  merged=$((merged + 1))

  echo "merge-env: $key взят из $local_file"
done < "$local_file"

echo "merge-env: подмешано ключей: $merged"
