-- name: GetLearningTopicPlanForUpdate :one
SELECT * FROM learning_topic_plans WHERE topic_id = $1 AND user_id = $2 FOR UPDATE;

-- name: EnsureLearningTopicPlan :exec
INSERT INTO learning_topic_plans (topic_id, user_id, daily_new, front, credit, credited_on, created_at, updated_at)
VALUES ($1, $2, sqlc.arg(daily_new)::smallint, 'translation', sqlc.arg(daily_new)::smallint::int, $3, $4, $4)
ON CONFLICT (topic_id) DO NOTHING;

-- name: SetLearningTopicPlanCredit :exec
UPDATE learning_topic_plans SET credit = $2, credited_on = $3, updated_at = $4 WHERE topic_id = $1;

-- name: SetLearningTopicPlanSettings :one
UPDATE learning_topic_plans SET daily_new = $3, front = $4, updated_at = $5
WHERE topic_id = $1 AND user_id = $2
RETURNING *;

-- Entries not yet in any set that can be shown: with the side the learner sees
-- first filled in, never the ones already known.
-- name: CountIntroducibleLearningEntries :one
SELECT count(*)::bigint FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND introduced_on IS NULL AND status <> 'known'
  AND (NOT sqlc.arg(need_translation)::boolean OR translation <> '');

-- name: IntroduceLearningEntries :execrows
UPDATE learning_entries SET introduced_on = sqlc.arg(day)::date
WHERE id IN (
    SELECT e.id FROM learning_entries e
    WHERE e.user_id = $1 AND e.topic_id = $2 AND e.introduced_on IS NULL AND e.status <> 'known'
      AND (NOT sqlc.arg(need_translation)::boolean OR e.translation <> '')
    ORDER BY e.created_at, e.id
    LIMIT sqlc.arg(take)::int
);

-- Today's set: what was introduced today, then what the learner did not know
-- before (which comes along with the new ones).
-- name: ListLearningSetToday :many
SELECT * FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known'
  AND (introduced_on = sqlc.arg(day)::date OR weak)
ORDER BY (introduced_on = sqlc.arg(day)::date) DESC, created_at, id;

-- Cards of a calendar week still to go through: introduced inside it and not yet
-- reviewed in it. Cards the learner did not know come first, the rest shuffled.
-- name: ListLearningWeekPending :many
SELECT * FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known'
  AND introduced_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
  AND (week_reviewed_on IS NULL OR week_reviewed_on < sqlc.arg(from_date)::date)
ORDER BY weak DESC, random()
LIMIT sqlc.arg(take)::int;

-- name: CountLearningWeek :one
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE week_reviewed_on IS NULL OR week_reviewed_on < sqlc.arg(from_date)::date)::bigint AS pending
FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known' AND introduced_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date;

-- name: ListLearningMonthPending :many
SELECT * FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known'
  AND introduced_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
  AND (month_reviewed_on IS NULL OR month_reviewed_on < sqlc.arg(from_date)::date)
ORDER BY weak DESC, random()
LIMIT sqlc.arg(take)::int;

-- name: CountLearningMonth :one
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE month_reviewed_on IS NULL OR month_reviewed_on < sqlc.arg(from_date)::date)::bigint AS pending
FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known' AND introduced_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date;

-- name: CountLearningSetToday :one
SELECT count(*) FILTER (WHERE introduced_on = sqlc.arg(day)::date)::bigint AS fresh,
       count(*) FILTER (WHERE weak AND introduced_on IS DISTINCT FROM sqlc.arg(day)::date)::bigint AS weak
FROM learning_entries
WHERE user_id = $1 AND topic_id = $2 AND status <> 'known' AND (introduced_on = sqlc.arg(day)::date OR weak);

-- How a card went in a set: known or not, and the date it was last reviewed in the
-- week / month, when the review was one of those.
-- name: GradeLearningSetEntry :one
UPDATE learning_entries
SET weak = $3,
    week_reviewed_on = CASE WHEN sqlc.arg(period)::text = 'week' THEN sqlc.arg(today)::date ELSE week_reviewed_on END,
    month_reviewed_on = CASE WHEN sqlc.arg(period)::text = 'month' THEN sqlc.arg(today)::date ELSE month_reviewed_on END
WHERE id = $1 AND user_id = $2
RETURNING *;
