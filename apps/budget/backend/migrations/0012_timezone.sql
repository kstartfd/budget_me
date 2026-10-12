-- +goose Up
-- Time was ambiguous: the client sent wall-clock date+time and the backend
-- parsed it as UTC, so a spend at 01:00 Minsk was stored as 01:00Z — three
-- hours off the real instant, and on the wrong calendar day near midnight.
--
-- From here on the backend is UTC-only: spent_at is a true instant, and the
-- CLIENT (the only party that reliably knows the zone) decides which day and
-- month that instant belongs to. Nothing about zones is stored here.
--
-- Existing rows carry Minsk wall-clock stamped as UTC (every account so far is
-- in Minsk, UTC+3, and Belarus has had no DST since 2011 — so a flat shift is
-- exact, not an approximation). Move them to the instant they actually happened.
UPDATE transactions SET spent_at = spent_at - interval '3 hours';

-- +goose Down
UPDATE transactions SET spent_at = spent_at + interval '3 hours';
