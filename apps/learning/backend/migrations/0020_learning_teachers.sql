-- +goose Up
-- A teacher is a language the learner studies, with the language its rules
-- are explained in. It exists before any test or program does, so the learner
-- can keep several languages side by side. A program, a test and a chat are
-- found by the teacher's language; one teacher per language matches the one
-- running program per language.
CREATE TABLE learning_teachers (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language text NOT NULL,
    translation_language text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (user_id, language)
);

-- Everyone who already studies a language gets its teacher, with the pair of
-- the newest program or test.
INSERT INTO learning_teachers (id, user_id, language, translation_language, created_at)
SELECT DISTINCT ON (user_id, language) gen_random_uuid(), user_id, language, translation_language, created_at
FROM (
    SELECT user_id, language, translation_language, created_at FROM learning_paths
    UNION ALL
    SELECT user_id, language, translation_language, created_at FROM learning_path_tests
) AS seen
ORDER BY user_id, language, created_at DESC;

-- +goose Down
DROP TABLE learning_teachers;
