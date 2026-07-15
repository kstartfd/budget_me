-- +goose Up
-- Actual money on hand. The app cannot know it from transactions alone (it only
-- sees what was logged, never what was already in the wallet), so the user
-- states it once: "right now I have X". That is the anchor — an amount plus the
-- moment it was true. The balance is then the anchor plus every transaction
-- since it, which also makes correcting the balance trivial: re-stating it just
-- moves the anchor to now, and older transactions stop counting.
--
-- Both columns are nullable: NULL = never stated, and the app asks instead of
-- showing a wrong zero.
ALTER TABLE users
    ADD COLUMN balance_anchor_minor bigint,
    ADD COLUMN balance_anchor_at    timestamptz;

-- +goose Down
ALTER TABLE users
    DROP COLUMN balance_anchor_minor,
    DROP COLUMN balance_anchor_at;
