#!/usr/bin/env bash
# Copy existing learning accounts and content into an empty, migrated learning DB.
# Keep the old database intact until the new app has been checked in Telegram.
set -euo pipefail

: "${SOURCE_DATABASE_URL:?set the existing budget database URL}"
: "${TARGET_DATABASE_URL:?set the new learning database URL}"
command -v psql >/dev/null || { echo 'psql is required' >&2; exit 1; }

if [[ "$SOURCE_DATABASE_URL" == "$TARGET_DATABASE_URL" ]]; then
  echo 'Source and target databases must differ' >&2
  exit 1
fi

existing=$(psql "$TARGET_DATABASE_URL" -XAtqc 'SELECT count(*) FROM users')
if [[ "$existing" != 0 ]]; then
  echo 'Target database must be empty; aborting without changes' >&2
  exit 1
fi

copy_dir=$(mktemp -d)
trap 'rm -rf "$copy_dir"' EXIT

# One read-only snapshot ensures the users, topics, entries, tokens and jobs match.
psql "$SOURCE_DATABASE_URL" -X -v ON_ERROR_STOP=1 <<SQL
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
\\copy (SELECT id, telegram_id, email, password_hash, first_name, username, language_code, created_at, updated_at FROM users WHERE id IN (SELECT user_id FROM learning_topics UNION SELECT user_id FROM learning_tokens UNION SELECT user_id FROM learning_captures)) TO '$copy_dir/users.csv' WITH (FORMAT csv, NULL '\\N')
\\copy (SELECT id, user_id, name, language, translation_language, created_at FROM learning_topics) TO '$copy_dir/topics.csv' WITH (FORMAT csv, NULL '\\N')
\\copy (SELECT id, user_id, topic_id, kind, text, transcription, translation, native_usage, example, rule, status, created_at, updated_at, explanation, source FROM learning_entries) TO '$copy_dir/entries.csv' WITH (FORMAT csv, NULL '\\N')
\\copy (SELECT id, user_id, name, prefix, token_hash, scope, created_at, last_used_at, revoked_at, topic_id FROM learning_tokens) TO '$copy_dir/tokens.csv' WITH (FORMAT csv, NULL '\\N')
\\copy (SELECT id, user_id, request_key, payload, status, attempts, available_at, locked_until, lease_id, entry_id, error, created_at, updated_at FROM learning_captures) TO '$copy_dir/captures.csv' WITH (FORMAT csv, NULL '\\N')
COMMIT;
SQL

# The target transaction rolls back completely if any import fails.
psql "$TARGET_DATABASE_URL" -X -v ON_ERROR_STOP=1 <<SQL
BEGIN;
\\copy users (id, telegram_id, email, password_hash, first_name, username, language_code, created_at, updated_at) FROM '$copy_dir/users.csv' WITH (FORMAT csv, NULL '\\N')
\\copy learning_topics (id, user_id, name, language, translation_language, created_at) FROM '$copy_dir/topics.csv' WITH (FORMAT csv, NULL '\\N')
\\copy learning_entries (id, user_id, topic_id, kind, text, transcription, translation, native_usage, example, rule, status, created_at, updated_at, explanation, source) FROM '$copy_dir/entries.csv' WITH (FORMAT csv, NULL '\\N')
\\copy learning_tokens (id, user_id, name, prefix, token_hash, scope, created_at, last_used_at, revoked_at, topic_id) FROM '$copy_dir/tokens.csv' WITH (FORMAT csv, NULL '\\N')
\\copy learning_captures (id, user_id, request_key, payload, status, attempts, available_at, locked_until, lease_id, entry_id, error, created_at, updated_at) FROM '$copy_dir/captures.csv' WITH (FORMAT csv, NULL '\\N')
COMMIT;
SQL

echo 'Learning data copied. Old database was not changed.'
