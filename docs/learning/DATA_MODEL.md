# Data model

Two parts: **A. current schema** (migrations `0001`-`0038`, read from the current backend tree) and
**B. proposed target schema** (everything in B is `PROPOSED` until its phase starts; see
`DECISIONS.md`). The source of truth for A is `apps/learning/backend/migrations/`; sqlc
generates `internal/db` from it (`sqlc.yaml`: `schema: migrations/`).

## 0. Conventions (apply to A and B)

- Learning-owned tables use descriptive names with the `learning_` prefix. The
  physical knowledge item table is `learning_knowledge_items` from migration 0038;
  `kb_items` remains an updatable compatibility view. `learner_item_state` and
  other unprefixed B-stage proposals need separate reconciliation before new
  migrations; see `IMPLEMENTATION_PLAN.md`.
- Primary keys: `uuid` generated in Go by `internal/ids` (time-ordered, UUIDv7-shaped). Exception: `learning_chat_messages.id bigserial`.
- Timestamps: `timestamptz`, set by the caller from `clock.Clock` (never `now()` in SQL), so tests control time.
- Ownership: every user-owned table carries `user_id` and **every query filters by it**. Child tables enforce it with a composite FK, e.g. `FOREIGN KEY (topic_id, user_id) REFERENCES learning_topics(id, user_id)` backed by `UNIQUE (id, user_id)`. New tables follow this pattern.
- Cascades: deleting a user cascades everything (`ON DELETE CASCADE`); this is how account deletion works.
- Migrations are goose files with `-- +goose Up/Down`, forward-compatible, one concern each. Enum types exist today (TD-07); **new code uses `text` + `CHECK`** so values can be added and removed (proposed D-009).
- Rule for B: *expand -> backfill -> switch reads -> contract*, each step its own deploy (every push to `main` deploys; see `IMPLEMENTATION_PLAN.md` section 4).

---

## A. Current schema

### A.1 Identity and settings

| Table | Purpose | Key columns / notes |
| --- | --- | --- |
| `users` | The app's own user row (copy of the shared account profile) | `id`, `telegram_id` unique, `email` unique, `password_hash`, `account_id` unique (-> `account_app` DB, no FK), names |
| `learning_ai_settings` | The learner's chosen agent | PK `user_id`; `provider` enum (anthropic, openai, deepseek, kimi), `model`, `level` enum, `key_ciphertext` (AES-256-GCM, bound to the row), `key_hint` (last 4), `teacher` bool |
| `learning_tokens` | Extension tokens | `token_hash` unique, `token_ciphertext` (revealable), `scope` (capture / read_write), optional `topic_id`, `revoked_at` |
| `learning_reminders` | Telegram reminder prefs | PK `user_id`, `timezone`, `last_sent_on` |
| `learning_chat_marks` | Highlight triggers for chat text | PK `user_id`, jsonb |
| `learning_agent_usage` | Token metering per user/hour/provider/model/purpose | PK `(user_id, hour, provider, model, purpose)`; `calls`, `input_tokens`, `cached_tokens`, `output_tokens` |

### A.2 Notebook universe (cards + SRS)

| Table | Purpose | Key columns / notes |
| --- | --- | --- |
| `learning_topics` | A notebook per language | `(id, user_id)` unique; `name`, `language`, `translation_language` |
| `learning_entries` | Word / phrase card | `topic_id`+`user_id` composite FK; `kind` (word/phrase), `text`, `transcription`, `translation`, `explanation`, `native_usage`, `example`, `rule`, `source`, `status` (inbox/learning/known); **SRS**: `review_stage smallint`, `next_review_at`, `last_reviewed_at`; **sets**: `introduced_on`, `weak`, `week_reviewed_on`, `month_reviewed_on`; `text_hash bytea` (SHA-256 of lower-cased, whitespace-folded text) with a **unique** index `(user_id, topic_id, text_hash)` |
| `learning_topic_plans` | Daily-set settings per notebook topic | PK `topic_id`; `daily_new` 1-50, `front` (translation/text), `credit`, `credited_on` |
| `learning_captures` | Durable capture queue (extension / app) | `request_key` unique per user (idempotency), `payload jsonb`, `status`, `attempts`, `available_at`, `locked_until`, `lease_id`, `entry_id`, `error` |
| `learning_scans` | Remembered pictures read by OCR | PK `(user_id, topic_id, image_hash)` |
| `learning_audio_cache` | TTS cache index | unique `(language, text_hash)`, `storage_path`, `content_hash` |

### A.3 Chat

| Table | Purpose | Notes |
| --- | --- | --- |
| `learning_chats` | One conversation | **Exactly one subject** (CHECK `num_nonnulls(entry_id, topic_id, path_topic_id, practice_id) = 1`): notebook entry, notebook topic (free teacher chat), path topic, or rule-practice sitting (+ `example_id`). `summary`, `summary_upto`, `context_tokens`, `busy_until` (one turn at a time), `spent_tokens`, `spent_since` (window budget), `archived_at` (closed dialogs; unique open chat per notebook topic), `scenario` |
| `learning_chat_messages` | Messages | `bigserial`, `role` user/assistant |
| `learning_chat_usage` | Daily token counters | PK `(scope user|key, subject, day)`; `subject` is the user id or an HMAC fingerprint of the API key |

### A.4 Teacher / program universe

```
users
  └─ learning_teachers (user, language) UNIQUE        explain_in_target
  └─ learning_path_tests (kind placement|final, status in_progress|done, level, level_reached, passed, path_id?)
       ├─ learning_path_test_questions (test, question_key) level, position, skill, text, prompt, options jsonb, answer
       └─ learning_path_test_answers   (test, question_key) answer, correct
  └─ learning_path_jobs (queued|running|done|failed; test_id; preferences; path_id)
  └─ learning_paths (user, language) status active|finished|abandoned; level; summary; source; preferences jsonb;
       │              placement_test_id (NO FK); UNIQUE one active per (user, language)
       └─ learning_path_topics (position, skill, unit, title, goal, theory, vocabulary jsonb, source_id TEXT, status)
            └─ learning_path_exercises (position, kind, text, prompt, options jsonb, answer, rule)
            │     └─ learning_path_exercise_versions (exercise PK; topic; generation int)
            │     └─ learning_path_attempts (exercise, user, answer, correct, mistakes, feedback, lesson_id?)
  └─ learning_path_lessons (status active|finished, started_at, ends_at, paused_*, topic_id NULL since 0025)
  └─ learning_rule_practice (user, rule_id) UNIQUE; examples jsonb, exercises jsonb; expires_at
```

Notes the next engineer must know:

- `learning_paths.placement_test_id` has **no foreign key**; `learning_path_tests.path_id` (final tests) is `ON DELETE CASCADE`.
- `learning_path_topics.source_id` is free text pointing into the embedded library (`a1-to-be`); empty for agent-written topics. No FK, no version.
- `learning_path_exercises.answer` is the key; **returned by the API** (G-07).
- Exercise "generation": the *current* set of a topic is the rows whose version has `MAX(generation)`; old generations and their attempts stay.
- `learning_path_attempts.mistakes` is a count (cloze gaps wrong / written-text errors), not a classification.
- `learning_path_exercise_kind` enum values: `choice, fill, translate` (0014), `write, listen, speak` (0022), `order, cloze, read` (0031).
- `learning_teachers.id` is **not** unique together with `user_id` yet (needed for composite FKs in B).

### A.5 Deletion semantics today

| Action | What is deleted |
| --- | --- |
| `DELETE /learning/topics/:id` | the notebook topic with its entries, chats and scans (cascade); capture receipts in `learning_captures` remain |
| `POST /path/reset` | nothing: the program is archived (`status != active`) |
| `DELETE /path/history?language=` | **hard delete** of closed programs and, per the query's own comment, everything under them: topics, exercises, attempts, lessons opened on those topics, topic chats and final tests — user-initiated, irreversible. Placement tests and their answers are *not* cascaded (`path_id` is NULL, `placement_test_id` has no FK) and remain; lessons with no topic remain. |
| `DELETE /path/teachers/:id` | the teacher; its running program is closed into history |
| Rule-practice leave | sitting and its chats after 10 min |
| Account deletion | after a grace period, everything of the user (shared-account sweep) |

---

### A.6 Learning loop (V1, migration `0036_learning_loop.sql`)

Added by the first vertical slice (`VERTICAL_SLICE_V1.md`); expand-only, one constraint
`learning_teachers_id_user UNIQUE (id, user_id)` is implied by the primary key.

| Table | Role | Notes |
| --- | --- | --- |
| `learning_knowledge_items` (formerly `kb_items`) | Identity of a piece of knowledge: `(language, key)` unique, `kind` (`grammar`/`topic`), `skill`, `cefr_level`, `unit` | Migration 0038 renames the same physical relation and keeps `kb_items` as a write-compatible view; no rows or FKs are copied. Text stays in the embedded library. The CHECKs are widened by P1. |
| `learning_evidence` | Append-only proof of what the learner showed | `source`, `practice_mode` (`tracked`/`assessment`; free practice writes nothing), `ref_type='attempt'` + `ref_id` (**no FK**, so clearing history keeps it), `mode`, `outcome`, `score`, `weight` (double precision), `assisted`, `attempt_no`, `hints_used`; `UNIQUE (ref_type, ref_id)`; composite FK to the teacher, cascade |
| `learner_item_state` | Derived state per teacher and item | `state`, recency-weighted sums, counters, `stage`, `lapses`, `last_success_at`, `due_at`; PK `(teacher_id, item_id)`; the review state is **not** stored |
| `learning_activities` | Typed learning activities | `type` (`learn/practice/review/apply/assess`), `status` (`open/completed/skipped`), `reason` (`path/due`), `goal`, `item_id`, `topic_id`, `params`; partial unique indexes: one open review per item, one open practice per topic |

Differences from the proposal in B.2: `source_type` became `source` + `practice_mode` + `ref_type`
(D-025/D-030); evidence has `assisted` instead of `created_by`; `mastery`/`confidence` columns are not
there (nothing in V1 presents a probability); state names follow D-028 on the wire. Deleting a
teacher cascades to all four tables; deleting program history (`DELETE /history`) does not touch them.

### A.7 Quick Screening (migration `0037_adaptive_assessment.sql`)

Opt-in assessment metadata extends `learning_path_tests` with a profile FK, type, lifecycle
state, policy snapshot, active batch and structured result. Existing test rows have null
assessment metadata. Questions snapshot an opaque version, canonical knowledge item ID and key,
objective, format and private grading key. Answers have a unique assessment attempt ID,
an outcome and an assistance flag. `learning_evidence.ref_type = 'assessment_attempt'`
links to that attempt ID. All state and outcome additions use text checks. Deleting a profile
removes its assessments and evidence, and clearing program history preserves them.

### A.8 Knowledge item table name (migration `0038_learning_knowledge_items_name.sql`)

`learning_knowledge_items` is the same relation and holds the same rows and UUIDs as
the original `kb_items`. Existing foreign keys follow the renamed relation. The
simple `kb_items` view supports the previous application's reads and
`INSERT ... ON CONFLICT` writes during blue-green deployment. Migration Down drops
the view and renames the same table back; it does not delete knowledge or evidence.

### A.9 Assessment section storage (migration `0039_assessment_sections.sql`)

`learning_assessment_sections` stores test ID, skill, order, lifecycle status, policy
and optional blueprint snapshots, current band/batch, answered count, stop reason,
section result and timestamps. `(test_id, skill)` and `(test_id, position)` are unique;
at most one section can be in progress, paused or evaluating per test. The table is
ready for multiple independent skill sections, but only the Quick grammar workflow is
exposed. `learning_assessment_section_questions` links a question to the section with
matching test and skill through composite FKs. Answers retain `(test_id, question_key)`;
`learning_assessment_section_evidence` joins them to the unique evidence attempt ID,
checks skill, user and teacher, and exposes the association without changing the
append-only evidence row. Unattributable historical evidence is not fabricated.

The migration backfills Quick as one grammar section. Its ID equals the test ID so it
survives a safe rollback and reapplication; original policy and results remain on the
legacy test. An old Quick row lacking a full blueprint or section result keeps NULL.
Triggers mirror writes by old and new binaries during rollout; new policy JSON also
contains the full section blueprint, ignored by old decoders. Down refuses independently
stored section data that cannot be reproduced from the legacy test. Full/Skill
orchestration, history and additional skill evaluators remain future work. See
[Assessment Modes](ASSESSMENT_MODES.md#5-storage-and-api-rollout).

## B. Proposed target schema (PROPOSED)

Design goals: (1) stable **knowledge item** identity independent of generated programs;
(2) one append-only **evidence** table every engine writes and reads; (3) **derived state**
(mastery, estimates) recomputable from evidence; (4) additive changes to existing tables,
no rewrite of the teacher/program universe.

The SQL sketches below are target shapes, not migration-ready DDL. Unprefixed
names such as new `learner_*`, `assessment_*`, and `study_*` tables are provisional.
The B.1 `CREATE TABLE learning_knowledge_items` block describes the intended
final fields; migration 0038 already supplies the physical
`learning_knowledge_items` table. P1 must extend that table, use descriptive
`learning_` names for new relations and preserve the old `kb_items` view while
older application versions may run. `learner_item_state` is still a table.
See [the P1 naming and rollout rule](IMPLEMENTATION_PLAN.md).

### B.1 Knowledge Base (Phase 1)

```sql
CREATE TABLE learning_knowledge_items (
    id            uuid PRIMARY KEY,
    language      text NOT NULL,              -- the studied language, BCP-47 base tag
    key           text NOT NULL,              -- stable slug, e.g. 'a1-to-be' (existing library id)
    kind          text NOT NULL CHECK (kind IN
                  ('grammar','lexeme','phrase','pronunciation','function','communication_task',
                   'reading_text','listening_task','speaking_task','writing_task')),
    skill         text NOT NULL CHECK (skill IN ('grammar','vocabulary','reading','writing','listening','speaking','pronunciation')),
    cefr_level    text NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2','C1','C2')),
    unit          text NOT NULL DEFAULT '',   -- grouping inside a level
    position      integer NOT NULL DEFAULT 0, -- authored order inside a level
    status        text NOT NULL CHECK (status IN ('draft','published','deprecated')),
    superseded_by uuid REFERENCES learning_knowledge_items(id),
    created_at    timestamptz NOT NULL,
    UNIQUE (language, key)
);

-- Content is per explanation language and versioned; an item's identity never changes.
CREATE TABLE learning_knowledge_item_versions (
    item_id              uuid NOT NULL REFERENCES learning_knowledge_items(id) ON DELETE RESTRICT,
    explanation_language text NOT NULL,
    version              integer NOT NULL,
    title                text NOT NULL,
    goal                 text NOT NULL,        -- can-do statement
    body                 jsonb NOT NULL,       -- theory, forms, examples, mistakes, vocabulary ...; schema per kind
    content_hash         bytea NOT NULL,
    change_note          text NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL,
    published_at         timestamptz,
    PRIMARY KEY (item_id, explanation_language, version)
);

CREATE TABLE learning_knowledge_item_relations (
    from_item uuid NOT NULL REFERENCES learning_knowledge_items(id) ON DELETE CASCADE,
    to_item   uuid NOT NULL REFERENCES learning_knowledge_items(id) ON DELETE CASCADE,
    relation  text NOT NULL CHECK (relation IN ('prerequisite','related','contrasts_with','part_of')),
    strength  smallint NOT NULL DEFAULT 1,      -- 1 hard / 2 soft for prerequisite
    PRIMARY KEY (from_item, to_item, relation),
    CHECK (from_item <> to_item)
);

CREATE TABLE learning_knowledge_categories (id uuid PRIMARY KEY, language text NOT NULL, slug text NOT NULL, parent_id uuid REFERENCES learning_knowledge_categories(id), UNIQUE (language, slug));
CREATE TABLE learning_knowledge_item_categories (item_id uuid REFERENCES learning_knowledge_items(id) ON DELETE CASCADE, category_id uuid REFERENCES learning_knowledge_categories(id) ON DELETE CASCADE, PRIMARY KEY (item_id, category_id));

CREATE TABLE learning_knowledge_sources (id uuid PRIMARY KEY, kind text NOT NULL CHECK (kind IN ('cefr_descriptor','textbook','corpus','authored','other')), citation text NOT NULL, url text NOT NULL DEFAULT '', license text NOT NULL DEFAULT '');
CREATE TABLE learning_knowledge_item_sources (item_id uuid REFERENCES learning_knowledge_items(id) ON DELETE CASCADE, source_id uuid REFERENCES learning_knowledge_sources(id), locator text NOT NULL DEFAULT '', PRIMARY KEY (item_id, source_id, locator));

-- Can-do objectives (CEFR descriptors or authored), many items serve one objective.
CREATE TABLE learning_knowledge_objectives (id uuid PRIMARY KEY, language text NOT NULL, cefr_level text NOT NULL, skill text NOT NULL, descriptor text NOT NULL, source_id uuid REFERENCES learning_knowledge_sources(id), UNIQUE (language, cefr_level, skill, descriptor));
CREATE TABLE learning_knowledge_item_objectives (item_id uuid REFERENCES learning_knowledge_items(id) ON DELETE CASCADE, objective_id uuid REFERENCES learning_knowledge_objectives(id) ON DELETE CASCADE, PRIMARY KEY (item_id, objective_id));
```

Import is **idempotent** from the embedded JSON by `(language, key)`; `learning_path_topics.source_id`
is backfilled into a new nullable `kb_item_id` (B.3). The JSON stays authoritative until a
content-admin flow exists (D-008). Prerequisite graph must be acyclic: enforced by an import-time check and a test, not by SQL.

### B.2 Learner profile and evidence (Phases 2, 5, 6)

The profile root **is** `learning_teachers` (a teacher = a language the learner studies), so
existing teacher profiles and history are preserved (D-007). Add `UNIQUE (id, user_id)` to it
to allow composite FKs.

```sql
ALTER TABLE learning_teachers
    ADD COLUMN goals jsonb NOT NULL DEFAULT '{}'::jsonb,        -- interests, variant, goal, domain; was per program
    ADD COLUMN weekly_minutes integer,                           -- study time budget (was path preferences.hoursPerWeek)
    ADD COLUMN session_minutes integer CHECK (session_minutes BETWEEN 5 AND 120),
    ADD COLUMN updated_at timestamptz;

CREATE TABLE learning_evidence (            -- APPEND-ONLY, the single record of "what the learner showed"
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id    uuid NOT NULL,
    item_id       uuid REFERENCES learning_knowledge_items(id),               -- NULL when not attributable yet
    skill         text NOT NULL,
    source_type   text NOT NULL CHECK (source_type IN ('assessment','exercise','review','session','tutor','self_report','manual')),
    source_id     uuid,                                        -- attempt / test answer / chat message ...
    mode          text NOT NULL CHECK (mode IN ('recognition','recall','production')),
    outcome       text NOT NULL CHECK (outcome IN ('correct','partial','incorrect')),
    score         real NOT NULL CHECK (score BETWEEN 0 AND 1),
    weight        real NOT NULL CHECK (weight BETWEEN 0 AND 1),  -- reliability of this evidence
    hints_used    smallint NOT NULL DEFAULT 0,
    latency_ms    integer,
    attempt_no    smallint NOT NULL DEFAULT 1,
    error_category text,                                       -- from a closed taxonomy (see PROGRESS_MODEL.md)
    created_by    text NOT NULL CHECK (created_by IN ('system','ai_validated','user')),
    occurred_at   timestamptz NOT NULL,
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);
CREATE INDEX learning_evidence_item ON learning_evidence(teacher_id, item_id, occurred_at DESC);
CREATE INDEX learning_evidence_time ON learning_evidence(teacher_id, occurred_at DESC);

CREATE TABLE learner_item_state (            -- DERIVED, recomputable from evidence (Phase 6)
    teacher_id uuid NOT NULL, user_id uuid NOT NULL,
    item_id    uuid NOT NULL REFERENCES learning_knowledge_items(id),
    state      text NOT NULL CHECK (state IN ('unseen','introduced','learning','practiced','mastered','relearning')),
    mastery    real NOT NULL DEFAULT 0 CHECK (mastery BETWEEN 0 AND 1),
    confidence real NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    stability_days real, interval_days real, due_at timestamptz,
    evidence_count integer NOT NULL DEFAULT 0, lapses integer NOT NULL DEFAULT 0,
    last_evidence_at timestamptz, updated_at timestamptz NOT NULL,
    PRIMARY KEY (teacher_id, item_id),
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);
CREATE INDEX learner_item_state_due ON learner_item_state(teacher_id, due_at) WHERE state <> 'unseen';

CREATE TABLE learner_skill_estimates (       -- DERIVED + history of assessments
    id uuid PRIMARY KEY, teacher_id uuid NOT NULL, user_id uuid NOT NULL,
    skill text NOT NULL, cefr_level text NOT NULL,
    score real NOT NULL, confidence real NOT NULL, evidence_count integer NOT NULL,
    basis text NOT NULL CHECK (basis IN ('assessment','practice','self_report','manual')),
    created_at timestamptz NOT NULL,
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);

CREATE TABLE learner_mistakes (              -- journal (Phases 6, 9); recurring errors are a query over this
    id uuid PRIMARY KEY, teacher_id uuid NOT NULL, user_id uuid NOT NULL,
    item_id uuid REFERENCES learning_knowledge_items(id), evidence_id uuid REFERENCES learning_evidence(id),
    category text NOT NULL, learner_text text NOT NULL, corrected_text text NOT NULL, note text NOT NULL DEFAULT '',
    resolved_at timestamptz, created_at timestamptz NOT NULL,
    FOREIGN KEY (teacher_id, user_id) REFERENCES learning_teachers(id, user_id) ON DELETE CASCADE
);
```

Evidence is **never updated or deleted** except by account/teacher deletion (cascade);
"never silently discard learning history". Estimates and item state may be recomputed.

### B.3 Changes to existing tables

| Phase | Table | Change |
| --- | --- | --- |
| 1 | `learning_path_topics` | `kb_item_id uuid NULL REFERENCES learning_knowledge_items(id)`; backfill from `source_id` + `language`; keep `source_id` for rollback |
| 3 | `learning_path_tests` | `purpose text` (`placement`, `final`, `skill`, `diagnostic`, `reassessment`), `target_skill text NULL`, `stop_reason text` (non-null `stop_reason` + `status='done'` = finished; `learner_stopped` = incomplete — **avoids adding a value to the `learning_path_test_status` enum and breaking the frontend's closed `z.enum`, R-07**); `level_reached` stays (compat) |
| 3 | `learning_path_test_questions` | `kb_item_id uuid NULL`, `objective_id uuid NULL`, `difficulty real NULL`, `generated_by text` |
| 3 | new `assessment_results` | per test per skill: `cefr_level, score, confidence, evidence_count` (also written to `learner_skill_estimates`) |
| 4 | `learning_paths` | `revision integer`; new `learning_path_revisions(path_id, revision, trigger, diff jsonb, created_at)` |
| 4 | `learning_path_topics` | `reason text` (`gap`,`goal`,`prerequisite`,`review`,`new`), `priority real`, `blocked boolean`, `skipped_at timestamptz` + `skip_reason text` (flags, **not** new values of `learning_path_topic_status`, R-07); new `learning_path_topic_items(topic_id, item_id, role)` for multi-item units |
| 5 | `learning_path_exercises` | `kb_item_id`, `schema_version smallint`, `difficulty real`, `generated_by text`; `topic_id` becomes NULLable *or* a new generalised `exercises` table (open question Q-03) |
| 5 | `learning_path_attempts` | `attempt_no`, `hints_used`, `latency_ms`, `mode`, `error_category`, `evidence_id` (every attempt writes one evidence row in the same transaction) |
| 7 | `learning_path_lessons` | `planned_minutes`, `active_seconds`, `plan jsonb`; new `study_session_items(session_id, position, kind, item_id, exercise_id, status, evidence_id)` |
| 8 | (optional) `learner_daily_activity` | rollup for weekly statistics, only if aggregate queries are too slow |

### B.4 Mapping existing data into the target

| Existing | Becomes |
| --- | --- |
| Library JSON topic | `learning_knowledge_items` (kind by skill: grammar -> `grammar`, vocabulary -> `lexeme` set, etc.) + `learning_knowledge_item_versions` per explanation language |
| Library `tags`, rule `related` | `learning_knowledge_item_relations` (`related`); prerequisites are **new authored content** (D-013) |
| Library `vocabulary[]` | child `lexeme` items `part_of` the topic item |
| `learning_teachers` | learner profile root |
| `learning_paths.preferences` | copied to `learning_teachers.goals` (latest program wins), still written to the program for history |
| `learning_path_test_answers` (+ questions) | `learning_evidence` (`source_type=assessment`, `mode=recognition`, `item_id` NULL until attributable) |
| `learning_path_attempts` | `learning_evidence` (`source_type=exercise`); `mode` from exercise kind (choice/listen = recognition; fill/order/cloze/speak = recall; translate/write/read = production) |
| `learning_entries` SRS fields | stay; optionally mirrored as `phrase` items later (D-010) |

Backfill is **idempotent and resumable** (`INSERT … ON CONFLICT DO NOTHING` keyed by
`source_type + source_id`), run as a migration-time SQL for small tables or an operator
command (`cmd/backfill-…`, like `cmd/migrate backfill-accounts`) for large ones.
