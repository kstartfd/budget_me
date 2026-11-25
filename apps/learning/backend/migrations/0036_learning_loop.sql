-- +goose Up
-- The first learning loop (docs/learning/VERTICAL_SLICE_V1.md): what the learner
-- showed (evidence), what that says about each piece of knowledge (item state),
-- when it is due again, and the typed activities that ask for it.
--
-- Expand-only: nothing that exists is changed except one unique constraint that
-- is implied by the primary key, so the previous release runs unchanged on this
-- schema. Every text column with a closed set of values is text + CHECK, never
-- a Postgres enum: an enum value cannot be taken back, and a value the
-- frontend's closed zod enums do not know breaks a screen.

-- The profile of a language (the teacher) can now be the target of composite
-- foreign keys, so a row of the loop can never belong to another learner's teacher.
ALTER TABLE learning_teachers ADD CONSTRAINT learning_teachers_id_user UNIQUE (id, user_id);

-- A canonical piece of knowledge. Identity only: its text stays in the embedded
-- library, which is still the source of what the learner reads. The row is made
-- from the library the first time the learner shows something about it.
-- `kind` and the level are widened by the knowledge-base phase (P1).
CREATE TABLE kb_items (
    id uuid PRIMARY KEY,
    language text NOT NULL,
    key text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('grammar', 'topic')),
    skill text NOT NULL CHECK (skill IN ('grammar', 'vocabulary', 'reading', 'writing', 'listening', 'speaking', 'pronunciation')),
    cefr_level text NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2')),
    unit text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    UNIQUE (language, key)
);

-- Append-only: a row is never updated or deleted, except with its teacher or
-- account. It points at what produced it (ref_*) without a foreign key, so
-- clearing the program history does not erase what the learner showed.
-- UNIQUE (ref_type, ref_id): one attempt gives at most one row, so writing it
-- twice is harmless.
--   source        where the exercise came from.
--   practice_mode how it was practised; free practice writes nothing, so it is
--                 not a value here.
--   mode          what the learner had to do: pick, recall a form, produce.
--   assisted      the answer key or a hint was in front of the learner.
CREATE TABLE learning_evidence (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id uuid NOT NULL,
    item_id uuid NOT NULL REFERENCES kb_items(id),
    skill text NOT NULL CHECK (skill IN ('grammar', 'vocabulary', 'reading', 'writing', 'listening', 'speaking', 'pronunciation')),
    source text NOT NULL CHECK (source IN ('rulebook', 'learning_path', 'study_session', 'review', 'ai_tutor', 'assessment')),
    practice_mode text NOT NULL CHECK (practice_mode IN ('tracked', 'assessment')),
    ref_type text NOT NULL CHECK (ref_type IN ('attempt')),
    ref_id uuid NOT NULL,
    mode text NOT NULL CHECK (mode IN ('recognition', 'recall', 'production')),
    outcome text NOT NULL CHECK (outcome IN ('correct', 'incorrect')),
    score double precision NOT NULL CHECK (score BETWEEN 0 AND 1),
    weight double precision NOT NULL CHECK (weight BETWEEN 0 AND 1),
    assisted boolean NOT NULL,
    attempt_no smallint NOT NULL DEFAULT 1 CHECK (attempt_no >= 1),
    hints_used smallint NOT NULL DEFAULT 0 CHECK (hints_used >= 0),
    occurred_at timestamptz NOT NULL,
    UNIQUE (ref_type, ref_id),
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);
CREATE INDEX learning_evidence_item ON learning_evidence(teacher_id, item_id, occurred_at DESC);
CREATE INDEX learning_evidence_time ON learning_evidence(teacher_id, occurred_at DESC);

-- Derived from the evidence, kept for reading and scheduling. No row = the
-- learner has shown nothing about the item (unknown). The review state (not
-- scheduled / scheduled / due / overdue) is never stored: it is read from due_at.
--   weighted_sum / weight_sum  recency-weighted score (30-day half-life); score = their ratio.
--   days                        distinct UTC days with evidence.
--   recall_successes            correct answers that were recall or production.
--   streak / streak_day         consecutive correct recall/production and the day it began.
--   stage                       consecutive spaced successes on the review ladder.
CREATE TABLE learner_item_state (
    teacher_id uuid NOT NULL,
    user_id uuid NOT NULL,
    item_id uuid NOT NULL REFERENCES kb_items(id),
    state text NOT NULL CHECK (state IN ('unseen', 'introduced', 'learning', 'practiced', 'mastered', 'relearning')),
    score double precision NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 1),
    weighted_sum double precision NOT NULL DEFAULT 0,
    weight_sum double precision NOT NULL DEFAULT 0,
    evidence_count integer NOT NULL DEFAULT 0,
    days integer NOT NULL DEFAULT 0,
    recall_successes integer NOT NULL DEFAULT 0,
    streak integer NOT NULL DEFAULT 0,
    streak_day date,
    stage smallint NOT NULL DEFAULT 0 CHECK (stage >= 0),
    lapses integer NOT NULL DEFAULT 0,
    last_evidence_at timestamptz,
    last_success_at timestamptz,
    due_at timestamptz,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (teacher_id, item_id),
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);
CREATE INDEX learner_item_state_due ON learner_item_state(teacher_id, due_at) WHERE due_at IS NOT NULL;

-- One concrete thing to do: its type, why it was chosen, what it is for, which
-- knowledge it touches, and whether it is done. Whether it is done does not
-- depend on any study session.
--   learn / apply / assess are part of the vocabulary and are not produced yet.
CREATE TABLE learning_activities (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id uuid NOT NULL,
    type text NOT NULL CHECK (type IN ('learn', 'practice', 'review', 'apply', 'assess')),
    status text NOT NULL CHECK (status IN ('open', 'completed', 'skipped')),
    reason text NOT NULL CHECK (reason IN ('path', 'due')),
    goal text NOT NULL DEFAULT '',
    item_id uuid REFERENCES kb_items(id),
    topic_id uuid REFERENCES learning_path_topics(id) ON DELETE CASCADE,
    params jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE,
    CHECK ((status = 'open') = (completed_at IS NULL) OR status = 'skipped')
);
CREATE UNIQUE INDEX learning_activities_one_open_review ON learning_activities(teacher_id, item_id)
    WHERE type = 'review' AND status = 'open';
CREATE UNIQUE INDEX learning_activities_one_open_practice ON learning_activities(teacher_id, topic_id)
    WHERE type = 'practice' AND status = 'open';
CREATE INDEX learning_activities_teacher ON learning_activities(teacher_id, status, created_at);

-- +goose Down
DROP TABLE learning_activities;
DROP TABLE learner_item_state;
DROP TABLE learning_evidence;
DROP TABLE kb_items;
ALTER TABLE learning_teachers DROP CONSTRAINT learning_teachers_id_user;
