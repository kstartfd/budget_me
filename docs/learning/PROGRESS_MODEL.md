# Progress model

Defines what the platform stores about learning, how derived numbers are computed, and the
rules that keep every displayed number honest. Companion to `LEARNING_METHODOLOGY.md`
(why) and `DATA_MODEL.md` B (tables). Status of the pieces: `IMPLEMENTATION_STATUS.md`.

## 1. Layers

```
raw facts (append-only)        learning_evidence            written by the backend only
   │  recomputable
   ▼
derived state                  learner_item_state           per (profile, item): state, mastery, due
                               learner_skill_estimates      per (profile, skill): CEFR band, score, confidence
   │  read-only
   ▼
read models / analytics        dashboard, review queue, weekly stats, insights   (no tables required)
```

Principle: **derived state can always be rebuilt from evidence** by a deterministic job
(`rebuild-learner-state`). If a formula changes, state is rebuilt; evidence never changes.

## 2. Today vs target

| Concern | Today | Target |
| --- | --- | --- |
| Raw facts | `learning_path_attempts` (answer, correct, mistakes, feedback), `learning_path_test_answers` (correct) | `learning_evidence` (+ backfill of both) |
| Item state | notebook cards only: `review_stage`, `next_review_at`, `status` | `learner_item_state` for KB items; notebook SRS unchanged |
| Skill level | one `level` string per test/program | per-skill estimate + confidence + basis |
| Time studied | `learning_path_lessons` (wall clock, pause) | `active_seconds` per session (Phase 7) |
| Program progress | topic status counts, exercises `{total, done}` | unchanged + item states |

## 3. Evidence record

Fields (see DDL in `DATA_MODEL.md` B.2): who (`user_id`, `teacher_id`), what (`item_id` nullable,
`skill`), how (`mode`: recognition/recall/production; `hints_used`, `latency_ms`, `attempt_no`),
result (`outcome`: correct/partial/incorrect, `score` 0..1, `weight` 0..1, `error_category`),
provenance (`source_type`, `source_id`, `created_by`: `system` | `ai_validated` | `user`), `occurred_at`.

Rules:

1. **Append-only.** No UPDATE, no DELETE (except account/teacher deletion cascade). A correction is a *new* row that references the old via `source_id` semantics, never an edit.
2. Written **in the same transaction** as the thing that caused it (attempt row, test answer row).
3. `weight` and `score` are computed by backend code from `mode`, `outcome`, `hints_used`, `created_by` using the table in `LEARNING_METHODOLOGY.md` section 2 — never taken from an AI response.
4. AI-originated evidence (`created_by = ai_validated`) is accepted only if it passes `AI_CONTRACTS.md` section 5 validation and is capped at weight 0.5.
5. `item_id` may be NULL for legacy rows and assessment answers not yet attributable; such rows count for **skill estimates** but not for **item state**.

### 3.1 Error taxonomy (closed list, PROPOSED)

`form` (wrong morphology), `agreement`, `tense_aspect`, `word_order`, `article_determiner`,
`preposition`, `lexical_choice`, `spelling`, `pronunciation`, `register`, `omission`,
`l1_interference`, `comprehension`, `other`. AI classification must pick from this list;
anything else is rejected and stored as `other`.

## 4. Item state computation

Inputs: all evidence rows of `(teacher, item)`. Output: `state`, `mastery`, `confidence`,
`stage/interval`, `due_at`, `lapses`, `evidence_count`.

- `mastery` = recency-weighted mean of `score` with weights `weight x 0.5^(age_days/30)`.
- `confidence` = `1 - exp(-W / 3)` where `W` is the sum of effective weights (≈ 0.63 after 3 full-weight pieces).
- `state` per the transition table in `LEARNING_METHODOLOGY.md` section 2.
- `due_at` per the scheduler (section 3 there).

All formulas are pure functions with table-driven tests and a property test
(`state` is a function of the evidence multiset order-independent except for the lapse sequence).

## 5. Skill estimates

Per `(teacher, skill)` and per assessment/practice basis:

- **Assessment basis** — discrete Bayesian posterior over CEFR levels (`ASSESSMENT_ENGINE.md` section 4). Stored: `cefr_level` = posterior mode, `score` = posterior probability of that level, `confidence` = `1 - normalised entropy`, `evidence_count`, `basis = assessment`.
- **Practice basis** — computed from item states: for each level `L`, coverage(`L`, skill) = share of published level-`L` items of that skill at `practiced` or better; estimate = highest `L` with coverage >= 0.7 and every lower level >= 0.6; `confidence` from the number of items with evidence. `basis = practice`.
- **Self-report / manual** — start points only (`basis = self_report|manual`), `confidence <= 0.2`, replaced as soon as assessment/practice evidence exists.

No overall level is stored. A summary figure, if shown, is the **weakest of the core
skills** (grammar, vocabulary, reading) and labelled so; skills with `confidence < 0.4` show
as "not enough data".

## 6. Analytics definitions (Phase 8)

Every metric below names its source and its denominator; below the minimum sample it renders
"not enough data" — never 0%, never a placeholder.

| Metric | Definition | Source | Min sample |
| --- | --- | --- | --- |
| Active study time (week) | sum of `active_seconds` of sessions in the ISO week, learner's timezone | sessions | 1 session |
| Streak days | consecutive local days with >= 1 evidence row | evidence | — |
| Review queue size | items with `due_at <= now` | item state | — |
| Retention rate | correct recalls / due reviews, last 30 days | evidence (`source_type=review`) | 20 reviews |
| Mastery coverage per level | mastered items / published items of that level, per skill | item state + KB | 10 items attempted |
| Skill estimate | latest `learner_skill_estimates` per skill + confidence + age | estimates | 1 assessment |
| Recurring errors | `error_category` x `item_id` with >= 3 incorrect in 30 days and not resolved | mistakes | 3 events |
| Words/phrases in notebook | `learning_entries` counts by status | notebook (exists) | — |
| AI tokens (admin/learner info) | `learning_agent_usage` sums | exists | — |

Insights are **rule-generated sentences with evidence references** (e.g. "Past Simple vs
Present Perfect: 4 mistakes in 10 days"), each carrying the ids it was computed from so the
UI can link to them. No AI-written insight is stored as fact; an AI may *phrase* a rule-made
insight but not originate one.

### 6.1 Empty and incomplete states (must be designed and tested)

| State | Behaviour |
| --- | --- |
| New learner, no assessment | dashboard shows the next action ("take a short check or choose a starting point"), no numbers |
| Assessment abandoned half-way | partial result stored as `incomplete`; skills with enough evidence show an estimate with low confidence, others "not enough data" |
| No AI agent | deterministic features work; AI-needing actions show the settings link (existing behaviour) |
| Skill never practised | no estimate; "not assessed yet" |
| Items with no attributable `item_id` (legacy) | counted in time/volume stats, excluded from mastery stats |

## 7. Validation of the model itself

- **Simulation**: synthetic learners with a known level and forgetting curve answer through the real handlers; the suite asserts estimate recovery and retention behaviour (see `TESTING_STRATEGY.md`).
- **Placement validity** (post-launch): compare the first placement estimate per skill with the practice-basis estimate after 4 weeks; report the disagreement distribution. Thresholds in `PRODUCT_VISION.md` section 6 are set from this.
- **Fakeness check**: a test enumerates every dashboard field and asserts it is derived from a query, not from constants or random values; fixtures must not use fabricated realistic-looking data in production seed paths.
