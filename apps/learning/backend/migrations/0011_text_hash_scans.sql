-- +goose Up
-- The hash of what an entry says, lowercased with its whitespace folded, so a
-- word or phrase met again (in a photo, say) is found without comparing text.
-- The application computes it the same way (learning/commands.TextHash); the
-- entries that exist are hashed here. The whitespace is spelled out, the set
-- Go's unicode.IsSpace has, because \s follows the database's locale and would
-- not fold a no-break space.
ALTER TABLE learning_entries ADD COLUMN text_hash bytea;
UPDATE learning_entries
SET text_hash = sha256(convert_to(lower(btrim(regexp_replace(text, '[\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+', ' ', 'g'))), 'UTF8'));
ALTER TABLE learning_entries ALTER COLUMN text_hash SET NOT NULL;
CREATE INDEX learning_entries_topic_hash ON learning_entries(user_id, topic_id, text_hash);

-- The pictures a topic has been read from, by the hash of the file, so the same
-- photo sent twice is recognized as one.
CREATE TABLE learning_scans (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id uuid NOT NULL,
    image_hash bytea NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, topic_id, image_hash),
    FOREIGN KEY (topic_id, user_id) REFERENCES learning_topics(id, user_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE learning_scans;
DROP INDEX learning_entries_topic_hash;
ALTER TABLE learning_entries DROP COLUMN text_hash;
