-- +goose Up
ALTER TABLE learning_path_tests
    ADD COLUMN assessment_profile_id uuid,
    ADD COLUMN assessment_type text CHECK (assessment_type IN
        ('quick_screening', 'full_assessment', 'skill_assessment', 'knowledge_diagnostic', 'periodic_reassessment')),
    ADD COLUMN assessment_status text CHECK (assessment_status IN
        ('created', 'in_progress', 'paused', 'evaluating', 'completed', 'partial', 'failed', 'cancelled')),
    ADD COLUMN start_band text,
    ADD COLUMN assessment_policy jsonb,
    ADD COLUMN assessment_explanation_language text,
    ADD COLUMN assessment_batch integer NOT NULL DEFAULT 0 CHECK (assessment_batch >= 0),
    ADD COLUMN assessment_result jsonb,
    ADD COLUMN assessment_stop_reason text,
    ADD FOREIGN KEY (assessment_profile_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE,
    ADD CHECK ((assessment_type IS NULL AND assessment_profile_id IS NULL AND assessment_status IS NULL)
        OR (assessment_type IS NOT NULL AND assessment_profile_id IS NOT NULL AND assessment_status IS NOT NULL
            AND start_band IS NOT NULL AND assessment_policy IS NOT NULL AND assessment_explanation_language IS NOT NULL
            AND kind = 'placement' AND path_id IS NULL
            AND (status = 'done') = (assessment_status IN ('completed', 'partial', 'failed', 'cancelled'))));

CREATE UNIQUE INDEX learning_assessments_one_open
    ON learning_path_tests (assessment_profile_id, assessment_type)
    WHERE assessment_type IS NOT NULL AND status = 'in_progress';

ALTER TABLE learning_path_test_questions
    ADD COLUMN assessment_item_id uuid REFERENCES kb_items(id),
    ADD COLUMN assessment_item_key text,
    ADD COLUMN assessment_category text,
    ADD COLUMN assessment_objective text,
    ADD COLUMN assessment_format text,
    ADD COLUMN assessment_version text,
    ADD COLUMN assessment_origin text CHECK (assessment_origin IN ('curated', 'ai')),
    ADD COLUMN assessment_fingerprint text,
    ADD COLUMN assessment_batch integer,
    ADD COLUMN assessment_alternatives jsonb,
    ADD COLUMN assessment_repeated boolean NOT NULL DEFAULT false;

CREATE UNIQUE INDEX assessment_questions_fingerprint
    ON learning_path_test_questions (test_id, assessment_fingerprint)
    WHERE assessment_fingerprint IS NOT NULL;

ALTER TABLE learning_path_test_answers
    ADD COLUMN assessment_attempt_id uuid UNIQUE,
    ADD COLUMN assessment_outcome text CHECK (assessment_outcome IN
        ('correct', 'partial', 'incorrect', 'skipped', 'evaluation_unavailable')),
    ADD COLUMN assessment_assisted boolean NOT NULL DEFAULT false;

ALTER TABLE learning_evidence DROP CONSTRAINT learning_evidence_ref_type_check;
ALTER TABLE learning_evidence DROP CONSTRAINT learning_evidence_outcome_check;
ALTER TABLE learning_evidence ADD CHECK (ref_type IN ('attempt', 'assessment_attempt'));
ALTER TABLE learning_evidence ADD CHECK (outcome IN ('correct', 'incorrect', 'partial', 'skipped', 'evaluation_unavailable'));

-- +goose Down
DELETE FROM learning_evidence WHERE ref_type = 'assessment_attempt';
DELETE FROM learning_path_tests WHERE assessment_type IS NOT NULL;
ALTER TABLE learning_evidence DROP CONSTRAINT learning_evidence_ref_type_check;
ALTER TABLE learning_evidence DROP CONSTRAINT learning_evidence_outcome_check;
ALTER TABLE learning_evidence ADD CHECK (ref_type IN ('attempt'));
ALTER TABLE learning_evidence ADD CHECK (outcome IN ('correct', 'incorrect'));
ALTER TABLE learning_path_test_answers
    DROP COLUMN assessment_attempt_id, DROP COLUMN assessment_outcome, DROP COLUMN assessment_assisted;
ALTER TABLE learning_path_test_questions
    DROP COLUMN assessment_item_id, DROP COLUMN assessment_item_key, DROP COLUMN assessment_category,
    DROP COLUMN assessment_objective, DROP COLUMN assessment_format, DROP COLUMN assessment_version,
    DROP COLUMN assessment_origin, DROP COLUMN assessment_fingerprint, DROP COLUMN assessment_batch,
    DROP COLUMN assessment_alternatives, DROP COLUMN assessment_repeated;
ALTER TABLE learning_path_tests
    DROP COLUMN assessment_profile_id, DROP COLUMN assessment_type, DROP COLUMN assessment_status,
    DROP COLUMN start_band, DROP COLUMN assessment_policy, DROP COLUMN assessment_explanation_language,
    DROP COLUMN assessment_batch, DROP COLUMN assessment_result, DROP COLUMN assessment_stop_reason;
