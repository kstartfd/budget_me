-- +goose Up
-- Building a program is a long run on the learner's own agent, so it runs in
-- the background: the request only queues a job and the screen follows it. A
-- job outlives the request, a closed tab and a dropped connection; one job per
-- test keeps a double tap from building two programs.
CREATE TYPE learning_path_job_status AS ENUM ('queued', 'running', 'done', 'failed');

CREATE TABLE learning_path_jobs (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    test_id uuid NOT NULL REFERENCES learning_path_tests(id) ON DELETE CASCADE,
    status learning_path_job_status NOT NULL,
    -- The survey the program is built around, kept so a retry builds the same.
    preferences jsonb NOT NULL DEFAULT '{}'::jsonb,
    -- The program the job made, once it exists.
    path_id uuid REFERENCES learning_paths(id) ON DELETE SET NULL,
    error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX learning_path_jobs_test ON learning_path_jobs(test_id, created_at DESC);

-- +goose Down
DROP TABLE learning_path_jobs;
DROP TYPE learning_path_job_status;
