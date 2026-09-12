-- name: ListTasksForDay :many
-- Everything that belongs on `day`: still-open work planned on or before it
-- (regardless of how old), plus whatever was ticked done ON that exact day.
SELECT * FROM tasks
WHERE user_id = sqlc.arg(user_id)
  AND (
    (done_on IS NULL AND planned_for <= sqlc.arg(day)::date)
    OR done_on = sqlc.arg(day)::date
  )
ORDER BY planned_for, created_at;

-- name: InsertTask :one
INSERT INTO tasks (
    id, user_id, title, note, planned_for, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $6
)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpdateTask :one
-- Full replace of the editable fields (title/note/planned_for) — this is also
-- what an explicit "отложить на завтра" is: a plain planned_for change, distinct
-- from the carry-over above, which never touches a row.
UPDATE tasks
SET title = sqlc.arg(title),
    note = sqlc.narg(note),
    planned_for = sqlc.arg(planned_for),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: SetTaskDone :one
UPDATE tasks
SET done_on = sqlc.narg(done_on),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);
