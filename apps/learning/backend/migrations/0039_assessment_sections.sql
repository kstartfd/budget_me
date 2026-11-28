-- +goose Up
-- Keep the existing test, question, answer and evidence row shapes intact: old
-- binaries can have prepared statements for them throughout a blue-green deploy.
CREATE TABLE learning_assessment_sections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id uuid NOT NULL REFERENCES learning_path_tests(id) ON DELETE CASCADE,
    skill text NOT NULL CHECK (skill IN ('grammar', 'vocabulary', 'reading', 'listening', 'writing', 'speaking', 'pronunciation')),
    position integer NOT NULL CHECK (position >= 0),
    status text NOT NULL CHECK (status IN ('created', 'in_progress', 'paused', 'evaluating', 'completed', 'partial', 'failed', 'cancelled')),
    blueprint_version text,
    policy_snapshot jsonb NOT NULL,
    blueprint_snapshot jsonb,
    current_band text NOT NULL,
    current_batch integer NOT NULL DEFAULT 0 CHECK (current_batch >= 0),
    answered integer NOT NULL DEFAULT 0 CHECK (answered >= 0),
    stop_reason text,
    result_snapshot jsonb,
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    UNIQUE (test_id, skill),
    UNIQUE (test_id, position),
    UNIQUE (id, test_id, skill)
);
CREATE UNIQUE INDEX learning_assessment_sections_one_active ON learning_assessment_sections (test_id)
    WHERE status IN ('in_progress', 'paused', 'evaluating');

-- A question belongs to the same test and skill as its section. Answers already
-- use (test_id, question_key); evidence uses the unique assessment_attempt_id.
CREATE UNIQUE INDEX learning_assessment_question_skill ON learning_path_test_questions (test_id, question_key, skill);
CREATE TABLE learning_assessment_section_questions (
    test_id uuid NOT NULL,
    question_key text NOT NULL,
    skill text NOT NULL,
    section_id uuid NOT NULL,
    PRIMARY KEY (test_id, question_key),
    FOREIGN KEY (test_id, question_key, skill)
        REFERENCES learning_path_test_questions(test_id, question_key, skill) ON DELETE CASCADE,
    FOREIGN KEY (section_id, test_id, skill)
        REFERENCES learning_assessment_sections(id, test_id, skill) ON DELETE CASCADE
);

-- Quick's parent row is the compatibility contract until the old release is
-- retired. Copy only what it actually recorded. Older attempts can lack a full
-- blueprint or a section result; NULL means unknown, never reconstructed.
-- +goose StatementBegin
CREATE FUNCTION learning_sync_quick_section() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.assessment_type = 'quick_screening' THEN
        INSERT INTO learning_assessment_sections
            (id, test_id, skill, position, status, blueprint_version, policy_snapshot, blueprint_snapshot,
             current_band, current_batch, answered, stop_reason, result_snapshot, created_at, completed_at)
        VALUES (NEW.id, NEW.id, 'grammar', 0, NEW.assessment_status, NEW.assessment_policy->>'version',
                NEW.assessment_policy, NEW.assessment_policy->'sectionBlueprint', NEW.level, NEW.assessment_batch,
                (SELECT count(*) FROM learning_path_test_answers WHERE test_id = NEW.id), NEW.assessment_stop_reason,
                (SELECT value FROM jsonb_array_elements(CASE WHEN jsonb_typeof(NEW.assessment_result->'sections') = 'array'
                    THEN NEW.assessment_result->'sections' ELSE '[]'::jsonb END) WHERE value->>'skill' = 'grammar' LIMIT 1),
                NEW.created_at, NEW.completed_at)
        ON CONFLICT (test_id, skill) DO UPDATE SET
            status = EXCLUDED.status, blueprint_version = EXCLUDED.blueprint_version,
            policy_snapshot = EXCLUDED.policy_snapshot, blueprint_snapshot = EXCLUDED.blueprint_snapshot,
            current_band = EXCLUDED.current_band, current_batch = EXCLUDED.current_batch,
            answered = EXCLUDED.answered, stop_reason = EXCLUDED.stop_reason,
            result_snapshot = EXCLUDED.result_snapshot, completed_at = EXCLUDED.completed_at;
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER learning_quick_section_sync AFTER INSERT OR UPDATE ON learning_path_tests
    FOR EACH ROW EXECUTE FUNCTION learning_sync_quick_section();

-- +goose StatementBegin
CREATE FUNCTION learning_link_quick_question() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO learning_assessment_section_questions (test_id, question_key, skill, section_id)
    SELECT NEW.test_id, NEW.question_key, NEW.skill, s.id
      FROM learning_assessment_sections s
      JOIN learning_path_tests t ON t.id = s.test_id
     WHERE s.test_id = NEW.test_id AND s.skill = NEW.skill AND t.assessment_type = 'quick_screening'
    ON CONFLICT (test_id, question_key) DO NOTHING;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER learning_quick_question_link AFTER INSERT ON learning_path_test_questions
    FOR EACH ROW EXECUTE FUNCTION learning_link_quick_question();

-- +goose StatementBegin
CREATE FUNCTION learning_count_quick_answers() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target_test uuid;
BEGIN
    target_test := CASE WHEN TG_OP = 'DELETE' THEN OLD.test_id ELSE NEW.test_id END;
    UPDATE learning_assessment_sections s SET answered = (
        SELECT count(*) FROM learning_path_test_answers a
        JOIN learning_assessment_section_questions q USING (test_id, question_key)
        WHERE q.section_id = s.id
    ) WHERE s.test_id = target_test AND s.skill = 'grammar'
      AND EXISTS (SELECT 1 FROM learning_path_tests t WHERE t.id = s.test_id AND t.assessment_type = 'quick_screening');
    RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER learning_quick_answer_count AFTER INSERT OR DELETE ON learning_path_test_answers
    FOR EACH ROW EXECUTE FUNCTION learning_count_quick_answers();

-- Install the bridge before backfill. PostgreSQL's DDL locks wait for old
-- transactions and prevent a write from falling between backfill and trigger.
INSERT INTO learning_assessment_sections
    (id, test_id, skill, position, status, blueprint_version, policy_snapshot, blueprint_snapshot,
     current_band, current_batch, answered, stop_reason, result_snapshot, created_at, completed_at)
SELECT t.id, t.id, 'grammar', 0, t.assessment_status, t.assessment_policy->>'version', t.assessment_policy,
       t.assessment_policy->'sectionBlueprint', t.level, t.assessment_batch,
       (SELECT count(*) FROM learning_path_test_answers a WHERE a.test_id = t.id), t.assessment_stop_reason,
       (SELECT value FROM jsonb_array_elements(CASE WHEN jsonb_typeof(t.assessment_result->'sections') = 'array'
           THEN t.assessment_result->'sections' ELSE '[]'::jsonb END) WHERE value->>'skill' = 'grammar' LIMIT 1),
       t.created_at, t.completed_at
FROM learning_path_tests t WHERE t.assessment_type = 'quick_screening'
ON CONFLICT (test_id, skill) DO NOTHING;

INSERT INTO learning_assessment_section_questions (test_id, question_key, skill, section_id)
SELECT q.test_id, q.question_key, q.skill, s.id FROM learning_path_test_questions q
JOIN learning_assessment_sections s ON s.test_id = q.test_id AND s.skill = q.skill
JOIN learning_path_tests t ON t.id = s.test_id AND t.assessment_type = 'quick_screening'
ON CONFLICT (test_id, question_key) DO NOTHING;

-- This view exposes the durable association without mutating append-only
-- evidence or inventing associations for historical records that lack a key.
CREATE VIEW learning_assessment_section_evidence AS
SELECT sq.section_id, sq.test_id, sq.question_key, sq.skill, a.assessment_attempt_id, e.id AS evidence_id, e.user_id
FROM learning_assessment_section_questions sq
JOIN learning_path_test_answers a USING (test_id, question_key)
JOIN learning_path_tests t ON t.id = sq.test_id
JOIN learning_evidence e ON e.ref_type = 'assessment_attempt' AND e.ref_id = a.assessment_attempt_id
    AND e.skill = sq.skill AND e.user_id = t.user_id AND e.teacher_id = t.assessment_profile_id;

-- +goose Down
-- Automatic rollback is safe for mirrored Quick rows only. Any future
-- independent section data must first be preserved outside this migration.
-- +goose StatementBegin
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM learning_assessment_sections s JOIN learning_path_tests t ON t.id = s.test_id
        WHERE t.assessment_type IS DISTINCT FROM 'quick_screening' OR s.skill <> 'grammar' OR s.position <> 0 OR s.id <> s.test_id
           OR s.status IS DISTINCT FROM t.assessment_status OR s.policy_snapshot IS DISTINCT FROM t.assessment_policy
           OR s.blueprint_version IS DISTINCT FROM t.assessment_policy->>'version' OR s.created_at IS DISTINCT FROM t.created_at
           OR s.blueprint_snapshot IS DISTINCT FROM t.assessment_policy->'sectionBlueprint'
           OR s.current_band IS DISTINCT FROM t.level OR s.current_batch IS DISTINCT FROM t.assessment_batch
           OR s.answered IS DISTINCT FROM (SELECT count(*) FROM learning_path_test_answers a
               JOIN learning_assessment_section_questions q USING (test_id, question_key) WHERE q.section_id = s.id)
           OR s.stop_reason IS DISTINCT FROM t.assessment_stop_reason OR s.completed_at IS DISTINCT FROM t.completed_at
           OR s.result_snapshot IS DISTINCT FROM (SELECT value FROM jsonb_array_elements(
               CASE WHEN jsonb_typeof(t.assessment_result->'sections') = 'array' THEN t.assessment_result->'sections' ELSE '[]'::jsonb END)
               WHERE value->>'skill' = 'grammar' LIMIT 1)
    ) THEN
        RAISE EXCEPTION 'independent assessment sections cannot be safely removed; preserve section data before rollback';
    END IF;
END $$;
-- +goose StatementEnd
DROP VIEW learning_assessment_section_evidence;
DROP TRIGGER learning_quick_answer_count ON learning_path_test_answers;
DROP FUNCTION learning_count_quick_answers();
DROP TRIGGER learning_quick_question_link ON learning_path_test_questions;
DROP FUNCTION learning_link_quick_question();
DROP TRIGGER learning_quick_section_sync ON learning_path_tests;
DROP FUNCTION learning_sync_quick_section();
DROP TABLE learning_assessment_section_questions;
DROP INDEX learning_assessment_question_skill;
DROP TABLE learning_assessment_sections;
