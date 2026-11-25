# Implementation plan

The master plan (owner's brief, 2026-11-25) turned into an executable, resumable plan for
this repository. **Live status is in `IMPLEMENTATION_STATUS.md`**; this file is the plan and
stays stable except when scope changes (record why in `DECISIONS.md`). Decisions that block
a phase are in `DECISIONS.md` section C.

Contents: 1. Map of the plan onto the existing product · 2. Gap analysis · 3. Delivery
rules · 4. Phases P0-P10 · 5. Prioritised backlog and critical path · 6. Risks.

Task IDs: `P<phase>-NN`. Sizes are rough (S <= half a day, M 1-2 days, L 3-5 days, XL > 1
week of focused work).

---

## 1. Plan domains -> what exists today

| Plan domain (section 6) | Existing implementation | Reuse / replace |
| --- | --- | --- |
| 6.1 Knowledge Base | embedded JSON library (5 pairs, A1-B2) + rule book | **reuse content**, replace storage/identity (P1) |
| 6.2 Learner Profile | `learning_teachers` + per-program preferences + AI settings | **extend** (P2); preserve teachers and history |
| 6.3 Assessment | AI-written ladder placement + AI examiner, final test, choose-level | **refactor**: backend measurement, AI for items (P3) |
| 6.4 Learning Path | AI picks 8-14 library topics; static | **extend** with prerequisites, gaps, revisions (P4) |
| 6.5 Exercise Engine | 9 kinds, persisted sets, attempts, deterministic + AI grading | **extend**: item links, evidence, rubric (P5) |
| 6.6 Adaptive Learning | Leitner SRS for notebook cards only | **new for KB items**; reuse the ladder (P6) |
| 6.7 Study Sessions | 30-min timer ("lesson") | **extend** into planned sessions (P7) |
| 6.8 Progress & Analytics | scattered scales/totals | **new aggregation** (P8) |
| 6.9 AI Tutor | 4 chat kinds, teacher mode, scenarios, DeepSeek-only streaming | **extend** with context + evidence (P9) |
| Advanced (Phase 10) | TTS, browser speech recognition, OCR exist | build on them (P10) |

## 2. Gap analysis

Gaps G-01..G-12 are defined in `CURRENT_STATE.md` section 10. Phase that closes each:

| Gap | Closed by | Notes |
| --- | --- | --- |
| G-01 canonical items | P1 | kb tables + import; `kb_item_id` on topics |
| G-02 / G-03 content coverage | P1 (structure), content work per D-013/D-021 | structure ships in P1; new content is a separate stream |
| G-04 thin profile | P2 | |
| G-05 AI-decided placement | P3 | shadow mode first |
| G-06 static program, completion != mastery | P4 + P6 | topic completion stays for UX; mastery is separate |
| G-07 weak attempt evidence, key exposure | P5 | frontend migration needed (hide keys) |
| G-08 SRS only for cards | P6 | |
| G-09 timer-only lessons | P7 | |
| G-10 no dashboard | P8 | |
| G-11 tutor without context, no mistake journal | P9 | |
| G-12 no roles | separate stream (D-012) | not in P0-P10 |

Migration requirements identified in the audit: (1) library import + backfill of
`source_id` -> `kb_item_id`; (2) replay of legacy test answers and attempts into evidence;
(3) enum-safe schema changes (D-009, R-07); (4) preserve all teachers/programs/history;
(5) keep extension API frozen (D-018).

## 3. Delivery rules (apply to every phase)

1. **Every push to `main` deploys production.** Land each phase as a sequence of small, green, backward-compatible pushes: *expand (schema) -> dual-write -> backfill -> switch reads (flag) -> contract*. Never combine a schema change and the code that requires it in a way the previous release cannot survive: two versions run during blue-green.
2. **Additive migrations only** in the expand step; drops/renames only in a later release after the code stopped using the column. Each migration has a working `Down`.
3. **Frontend-first for new enum values** (R-07): ship tolerant schemas one release before the backend emits anything new.
4. **Flags** (D-014) gate every behaviour switch; default = old behaviour.
5. **Tests with the change** (owner's rule) and the `VERIFIED` definition in `TESTING_STRATEGY.md` section 7. AI tests use fakes only.
6. **Docs in the same commit**: `DATA_MODEL`, `API_CONTRACTS`, `AI_CONTRACTS`, `IMPLEMENTATION_STATUS`, plus `learning-api.md` for endpoint changes.
7. **No destructive migration** without a restorable backup (the deploy already backs up) and an explicit owner "go" (D-002/D-003).
8. **One phase `IN_PROGRESS` at a time**, except pure-logic packages explicitly marked parallelisable.

---

## 4. Phases

### P0 — Repository audit and baseline

**Objective.** Know exactly what exists, record it where the next agent will find it, establish a verified baseline, and surface the decisions that gate the work.

**Current implementation.** `CURRENT_STATE.md` (audit of commit `4662ad7`), baseline: 262 backend tests pass, 2 skip; vet/eslint/tsc clean.

**Required changes.** Documentation only, plus a few hygiene tasks that do not change behaviour.

**Dependencies.** None. Owner answers for D-001..D-005, D-020.

**Database changes.** None.

**API changes.** None.

**Implementation tasks.**
- P0-01 Audit repository, stack, architecture, AI integration, tests — documented in `CURRENT_STATE.md`, `ARCHITECTURE.md` (S, done)
- P0-02 Record schema (migrations 0001-0035) and API contracts — `DATA_MODEL.md` A, `API_CONTRACTS.md` 1-2 (S, done)
- P0-03 Map to target architecture, gaps, backlog, decisions — this file, `DECISIONS.md` (M, done)
- P0-04 Run baseline gates and record them — `CURRENT_STATE.md` section 9 (S, done)
- P0-05 Make DB tests re-runnable (clean `learning_chat_usage` per test or fixture key per test) (S)
- P0-06 Frontend contract-test harness: zod schemas parse golden backend JSON; tolerate unknown enum values (M; needs D-019)
- P0-07 Migration test harness: up -> down -> up with seeded legacy rows (M)
- P0-08 Coverage report in CI, non-gating (S)
- P0-09 Read-only SQL report of production shape (row counts, largest users, test/program counts) for the owner to run (S; D-020)
- P0-10 Owner resolves D-001..D-005, D-020 (owner)
- P0-11 **Frontend enum tolerance release** (R-07): make every `z.enum` that the backend may extend tolerant (`.catch`/fallback) and unit-test it (S-M; unblocks P3/P4 safely)

**Acceptance criteria.** All 15 documents exist and cross-reference; every existing endpoint is indexed; baseline numbers recorded with commands; decision list complete with options and recommendation; no production behaviour changed.

**Testing requirements.** Baseline run repeatable on a fresh DB; P0-05 proves a second run on the same DB passes; P0-11 adds schema tests.

**Migration strategy.** n/a.

**Current status.** Documentation tasks `IMPLEMENTED`; P0-04 `VERIFIED` (2026-11-25); P0-05..P0-11 `NOT_STARTED`. Phase: `IN_PROGRESS` until D-001..D-005 are answered.

**Known limitations.** Production was not inspected (R-01). Playwright e2e not run. Baseline was taken with Go 1.27.4 locally while CI uses 1.25.

---

### P1 — Knowledge Base

**Objective.** A canonical, versioned, relational knowledge model with stable item ids, replacing the embedded-JSON-only library as the runtime source, **without changing what learners see**.

**Current implementation.** Embedded JSON, topic-level, ids unique per file, no relations/versions/sources; rule book served from it; program topics reference it by free-text `source_id` (`KNOWLEDGE_BASE.md` section 1).

**Required changes.** `kb_*` tables; idempotent importer; DB-backed read path behind a flag with a parity test; backfill `kb_item_id`; categories from `unit`; `related` relations from rule extras; `authored` source; read endpoints; generalise library validation tests to every pair; (per D-010) lexeme items or defer.

**Dependencies.** D-002/D-003 (owner's "go"), **D-008, D-009, D-010, D-014, D-018, D-020**; recommended before it: P0-05, P0-06, P0-07, P0-11.

**Database changes.** `DATA_MODEL.md` B.1 (all new tables) + `learning_path_topics.kb_item_id` (+ `kb_version`). No existing column changed.

**API changes.** New session-only `GET /learning/kb/items`, `/kb/items/:id`, `/kb/categories`. Rule-book endpoints byte-identical.

**Implementation tasks.**
- P1-01 JSON Schemas for `body` per kind (grammar/rule first) and generalise `curriculum_test` field checks to all five pairs (M)
- P1-02 Migration `kb_*` tables (expand only) with Down (S)
- P1-03 `learningkb` package: sqlc queries, `Importer` (idempotent, per-item transaction, content-hash versioning, report) (L)
- P1-04 Import topics of all pairs as items + versions per explanation language; assert counts (85/91 topics) (M)
- P1-05 Import `related`/tags -> relations; categories from `unit`; `authored` source (M)
- P1-06 `curriculum` read interface with embedded and DB implementations; config flag `LEARNING_KB_READ=embedded|db|shadow` (M)
- P1-07 Parity test: every rule/topic/word of every pair identical between embedded and DB (M)
- P1-08 Shadow mode: serve embedded, compare with DB in the background, log diffs (S)
- P1-09 Backfill `learning_path_topics.kb_item_id` from `source_id` (+language); orphan report; idempotent (M; needs P0-09 shape)
- P1-10 KB read endpoints + authorization tests (M)
- P1-11 Operator command `cmd/kb-import` and deploy hook (S)
- P1-12 Docs: `KNOWLEDGE_BASE`, `DATA_MODEL`, `API_CONTRACTS`, `learning-api.md` (S)
- P1-13 (later release) Contract: drop the embedded read path after one clean release on `db` (S)

**Acceptance criteria.**
- Importing into an empty DB creates exactly 85 (English) / 91 (Spanish) topic items per language and the stated rules; re-import changes nothing (0 added/changed).
- With `LEARNING_KB_READ=db` every `GET /path/rules*` and every program-build library read equals the embedded result (parity test green for all five pairs).
- Every existing `learning_path_topics.source_id` that is non-empty maps to an item or appears in the orphan report; zero silent drops.
- Prerequisite graph tooling exists (acyclicity/level checks) and passes on the (still empty) graph.
- No change to any existing response body (golden test).
- Authorization matrix passes for KB endpoints.

**Testing requirements.** Unit (hashing, versioning), DB (constraints, uniqueness), import idempotency, parity, migration up/down/up, backfill idempotency, HTTP + authz, gold JSON of existing endpoints unchanged.

**Migration strategy.** Expand -> import -> shadow compare -> flip `db` -> one release -> contract. Backfill is a resumable operator command; counts logged before/after.

**Current status.** `NOT_STARTED`.

**Known limitations.** No prerequisites (authored in P4 per D-013); no C1/C2; vocabulary lexeme items depend on D-010; no content admin UI (D-008).

---

### P2 — Learner Profile

**Objective.** One independent profile per `(user, studied language)` with goals, preferences, time, skill estimates, item proficiency and an evidence ledger — populated from existing history without losing any.

**Current implementation.** `learning_teachers` (language + explanation language); preferences stored per program; no estimates, no item state, no evidence.

**Required changes.** Extend the teacher row (D-007); create `learning_evidence`, `learner_skill_estimates`, `learner_item_state` (state computation only; scheduling in P6); dual-write evidence from attempt/test-answer handlers; replay legacy history; profile endpoints; survey prefilled from the profile.

**Dependencies.** P1 (items for attribution), **D-007, D-009, D-011**.

**Database changes.** `DATA_MODEL.md` B.2: `UNIQUE (id, user_id)` on `learning_teachers`, new columns, three new tables, indexes.

**API changes.** `GET /learning/profiles`, `GET|PUT /learning/profiles/:language`; teacher responses gain a `profile` summary.

**Implementation tasks.**
- P2-01 Migration: teacher columns + `UNIQUE(id,user_id)` (S)
- P2-02 Migration: `learning_evidence`, `learner_skill_estimates`, `learner_item_state` (S)
- P2-03 `learningprofile` package: `RecordEvidence` (weights/modes per methodology), pure `ApplyEvidence` (state, mastery, confidence) with tests (L)
- P2-04 Dual-write: attempt handler and test-answer handler write evidence in the same transaction; attribute attempts to the topic's `kb_item_id` (coarse) (M)
- P2-05 Backfill by replay: `learning_path_test_answers` -> evidence + posterior-free legacy estimates (`basis=assessment`, flagged legacy); `learning_path_attempts` -> evidence; idempotent by `(source_type, source_id)` (L; needs D-020)
- P2-06 `rebuild-learner-state` command (M)
- P2-07 Profile endpoints; survey prefill; preferences written to profile and program (M)
- P2-08 Authorization matrix (profile of language X vs Y, other user) (S)
- P2-09 Docs (S)

**Acceptance criteria.** Every existing teacher has a profile row (same id) with no data loss; evidence count after backfill equals `count(test_answers) + count(attempts)` of the migrated rows (verified query); re-running backfill adds 0; item state is a pure function of evidence (property test); no learner-visible behaviour changed; deleting a user/teacher cascades evidence.

**Testing requirements.** Unit (weights, state transitions, confidence), DB (composite FKs, cascade), backfill idempotency + count equality, same-transaction atomicity test (failure injected between attempt and evidence rolls both back), authz matrix, migration up/down/up.

**Migration strategy.** Expand -> dual-write on -> backfill -> verify counts -> (nothing reads yet; P3/P6 read) -> no contract step needed.

**Current status.** `NOT_STARTED`.

**Known limitations.** Attribution of attempts is topic-level until P5; assessment answers have no `item_id`; estimates from legacy tests are coarse and labelled `legacy`.

---

### P3 — Assessment Engine

**Objective.** Backend-owned, adaptive, evidence-based assessment per skill with confidence, partial results and manual starting points; the AI writes items and explanations but never decides a result.

**Current implementation.** `ASSESSMENT_ENGINE.md` section 1.

**Required changes.** Pure posterior engine, selection and stop rules; item source abstraction (curated bank + AI) with `assess.items/v2`; per-answer evidence; shadow mode vs the AI examiner; flag-switch; skill-specific and start-point inputs; incomplete/resume; knowledge-gap analysis; item analysis.

**Dependencies.** P1, P2, **D-009, D-014, D-017, D-022**, R-07 release (P0-11).

**Database changes.** `DATA_MODEL.md` B.3 rows for `learning_path_tests` (`purpose`, `target_skill`, `stop_reason`), `learning_path_test_questions` (`kb_item_id`, `objective_id`, `difficulty`, `generated_by`), new `assessment_results`, item-stats columns/table.

**API changes.** `POST /path/tests` optional `purpose/skills/startPoint`; responses add `progress`, `stopReason`; `POST /path/tests/:id/stop`; assessment history endpoints; `GET /path/tests/:id` returns per-skill progress. All additive.

**Implementation tasks.**
- P3-01 Pure package `learningassess/engine`: posterior update, expected-information selection, stop rules (L)
- P3-02 Simulation harness: synthetic learners, acceptance statistics (M)
- P3-03 Migration for tests/questions/results (S)
- P3-04 Item source interface; curated bank seeding from rule `forms/examples/mistakes` (M); AI source with `assess.items/v2` + validators + golden tests (M)
- P3-05 Per-answer evidence + per-skill results/estimates in the completion transaction (M)
- P3-06 Shadow mode: backend verdict stored beside the AI verdict; disagreement log (M)
- P3-07 Flag `LEARNING_ASSESS_ENGINE=ai|shadow|backend`; switch path incl. final test and legacy `levelReached` (D-022) (M)
- P3-08 Start point / skill-specific / diagnostic purposes (M)
- P3-09 Stop/incomplete/resume semantics (M)
- P3-10 Gap analysis query + feed to `Analyze` (replaces wrong-answer list) (M)
- P3-11 Item analysis and quarantine of suspect items (M)
- P3-12 Reassessment trigger predicates (backend only) (S)
- P3-13 Docs + removal plan for `test-judge` (S)

**Acceptance criteria.** The criteria in `ASSESSMENT_ENGINE.md` section 8; shadow data collected over >= 30 real or simulated tests with the agreed agreement metric before the flag flips; old clients still complete a test end-to-end; `levelReached` populated per D-022.

**Testing requirements.** Pure-unit + property tests of the engine; simulation (seeded) with assertion thresholds; AI contract golden tests; HTTP integration (full test run on the fake provider, both flag modes); authz matrix; migration up/down/up.

**Migration strategy.** `ASSESSMENT_ENGINE.md` section 7 (frontend tolerance -> expand -> replay backfill -> shadow -> switch -> contract).

**Current status.** `NOT_STARTED`.

**Known limitations.** Listening/speaking items arrive in P10; free-form scoring evidence weight capped; curated bank starts small (seeded from rules).

---

### P4 — Learning Path Engine

**Objective.** Paths generated from canonical items using prerequisites, ranked gaps, goals and review, valid by construction, updated dynamically while preserving history; buildable without an AI agent.

**Current implementation.** `LEARNING_ENGINE.md` section 1.

**Required changes.** Prerequisite data and validation; deterministic builder + AI shaping (`path.shape/v1`); priority function; `AdjustPath` with revisions; goal tasks; review activities; additive API.

**Dependencies.** P1, P2, P3 (gaps), **D-013, D-014, D-022**.

**Database changes.** `DATA_MODEL.md` B.3 rows for `learning_paths` (`revision`), `learning_path_revisions`, `learning_path_topics` (`reason`, `priority`, `blocked`, `skipped_at`, `skip_reason`), `learning_path_topic_items`.

**API changes.** `GET /path` additive fields; `GET /path/revisions`; `POST /path/refresh`. No new enum values (flags instead).

**Implementation tasks.**
- P4-01 Author/import prerequisite relations for English and Spanish (content stream, D-013) (XL, content)
- P4-02 `ValidatePath` + graph checks (acyclic, level-monotone) wired into KB import tests (M)
- P4-03 `Priority` function and config (M)
- P4-04 Deterministic builder using gaps (P3) + prerequisites; no AI (L)
- P4-05 `path.shape/v1` AI shaping with fallback (M)
- P4-06 Migration + revisions + flags on topics (S)
- P4-07 `AdjustPath` (insert/defer/skip/re-rank/extend) with bounds + revision log (L)
- P4-08 Goal-based `communication_task` items in the path (M; needs content)
- P4-09 Review activities (`reason=review`) (M)
- P4-10 Switch `Analyze` to the new builder behind `LEARNING_PATH_ENGINE` (M)
- P4-11 Endpoints + docs (S)

**Acceptance criteria.** Every generated path passes `ValidatePath` (property test over random profiles on the real KB); a path builds with no agent configured; `AdjustPath` is idempotent and never alters completed/in-progress topics or deletes history (test); a revision row explains every change; old `GET /path` consumers unaffected (golden).

**Testing requirements.** Property tests on random learner states; golden path for fixed profiles; DB tests for revisions; HTTP + authz; migration; AI contract tests; regression: library-less pair still builds via the AI fallback.

**Migration strategy.** Flag-gated switch per `LEARNING_PATH_ENGINE`; running programs are untouched (adjustments only on request/trigger, bounded).

**Current status.** `NOT_STARTED`.

**Known limitations.** Without prerequisite data the validator is vacuous; goal tasks need authored `communication_task` content; non-library pairs keep the AI-written fallback.

---

### P5 — Exercise Engine

**Objective.** Standardised, validated exercises linked to knowledge items; every attempt becomes evidence with mode, hints and error category; key exposure fixed; free-form answers scored by rubric.

**Current implementation.** `CURRENT_STATE.md` section 5.4.

**Required changes.** `exercises/v2` contract and `schema_version`; item linkage in generation and backfill; attempts -> evidence in one transaction; `reveal` endpoint and hidden keys; rubric scoring (C-2); mistake classification and journal write; wider deterministic grading; decide exercise table generalisation (Q-03).

**Dependencies.** P1, P2, (P3 for rubric contract reuse), **D-009**, frontend change for hidden keys.

**Database changes.** `DATA_MODEL.md` B.3 P5 rows; `learner_mistakes` (P2/P6 shared).

**API changes.** Additive attempt request/response fields; `POST /exercises/:id/reveal`; `X-Hide-Keys` opt-in, default flipped in a later release.

**Implementation tasks.**
- P5-01 Exercise schema v2 + contract + validators (extend `validExercise`) + golden tests (M)
- P5-02 Item-aware generation prompts (topic items listed; `itemId` validated) and backfill of existing exercises to the topic's item (M)
- P5-03 Attempt -> evidence in the same transaction; `attempt_no`, `hints_used`, `latency_ms` (M)
- P5-04 `reveal` endpoint, hide-keys flag, frontend change to call it (M, FE+BE)
- P5-05 Rubric scoring contract C-2, clamping, weight caps; replaces `review` correctness for translate/write/read (L)
- P5-06 Error taxonomy, classification (deterministic first, AI second) and `learner_mistakes` writes (M)
- P5-07 Deterministic grading extensions (accepted variants, accent-insensitive Spanish option, numerals) with tests per kind (M)
- P5-08 Exercise generalisation for review use (Q-03): topic-less exercises per item (M)
- P5-09 Docs + `learning-api.md` (S)

**Acceptance criteria.** Every new exercise has an `itemId` in its topic's item set; every attempt row has exactly one evidence row (verified by query and a rollback test); with `X-Hide-Keys: 1` no response contains an answer key before an attempt; a rubric reply with out-of-range scores is clamped and one with unknown categories is reduced to `other` (tests); existing clients unaffected.

**Testing requirements.** Contract golden tests, per-kind deterministic grading tables, transaction rollback tests, HTTP + authz, FE contract test for the new fields, migration + backfill idempotency.

**Migration strategy.** Expand columns -> dual-write -> backfill old exercises -> flag hidden keys opt-in -> frontend migrates -> flip default (two releases) -> contract.

**Current status.** `NOT_STARTED`.

**Known limitations.** Rubric agreement with humans unmeasured (weight cap stays); speaking exercises remain similarity-based until P10.

---

### P6 — Adaptive Learning

**Objective.** Mastery states, review scheduling, mistake tracking, recurring-error detection and learning priorities computed by the backend from evidence, adapting future paths.

**Current implementation.** Notebook-only Leitner (`NextReview`), daily sets, `weak` flag.

**Required changes.** Full item state machine; scheduler on the stage ladder; review queue; recurring-error detection; `Priority`; `AdjustPath` triggers; rebuild job; simulation tests; (D-010) decide notebook relation.

**Dependencies.** P2, P4 (AdjustPath), P5 (item-linked evidence), **D-010, D-016**.

**Database changes.** Uses `learner_item_state` (`due_at` index), `learner_mistakes`; no new tables unless D-010(b).

**API changes.** `GET /profiles/:language/review-queue`, `GET /mistakes`.

**Implementation tasks.**
- P6-01 `NextReview` for items (pure) incl. mode/hint rules, table tests (M)
- P6-02 State machine transitions per `LEARNING_METHODOLOGY` section 2 (M)
- P6-03 Update item state in the evidence transaction (M)
- P6-04 Review queue query + endpoint (M)
- P6-05 Mistake journal, recurring detection, resolution (M)
- P6-06 `AdjustPath` triggers (lapse cluster, recurrence) (M)
- P6-07 Simulation: synthetic forgetting curves, retention assertions (M)
- P6-08 Rebuild job equivalence test (state from scratch == incremental) (S)
- P6-09 Notebook relation per D-010 (S-XL)
- P6-10 Docs (S)

**Acceptance criteria.** A learner who answers correctly at increasing delays reaches `mastered` only after a >= 7-day-delay success; a lapse moves a `practiced/mastered` item to `relearning` and schedules it; incremental state equals rebuilt state on random histories (property test); the queue contains only due items and is bounded; recurring-error rule matches its definition on fixtures.

**Testing requirements.** Property + simulation + DB + HTTP + authz.

**Migration strategy.** State computed for existing evidence by `rebuild-learner-state`; scheduler gated by flag until validated on real data.

**Current status.** `NOT_STARTED`.

**Known limitations.** Parameters are untuned defaults; ladder scheduler (not FSRS) per D-016.

---

### P7 — Study Session Engine

**Objective.** Configurable-duration, balanced, resumable sessions that combine new material, review and application, with real active-time tracking.

**Current implementation.** `learning_path_lessons`: 30-minute server timer, pause/resume, attempts tagged with the lesson, no content planning.

**Required changes.** `PlanSession`; plan persistence; pre-generation of exercises; resume; heartbeat-based active time; session-end hooks; endpoints.

**Dependencies.** P4, P5, P6.

**Database changes.** `DATA_MODEL.md` B.3 P7 (`planned_minutes`, `active_seconds`, `plan`) + `study_session_items`.

**API changes.** `POST /path/lessons` accepts `minutes` and returns `plan`; `GET /lessons/active` returns plan + cursor; heartbeat and skip endpoints.

**Implementation tasks.**
- P7-01 Migration (S) · P7-02 `PlanSession` pure + tests (L) · P7-03 Pre-generation and deterministic fallback (M) · P7-04 Persist/resume (M) · P7-05 Heartbeat, clamp, idle handling (M) · P7-06 Session-end hook -> `AdjustPath`, summary (M) · P7-07 Endpoints + docs (S)

**Acceptance criteria.** Planned slices respect the 40/35/25 composition within tolerance for any pool; no item repeated within a session; resumed sessions continue at the cursor; forgetting a tab open adds at most 30 s per heartbeat window (test); without an agent the plan contains only deterministic kinds; the old lesson endpoints behave as before when called without new fields.

**Testing requirements.** Pure planning tests (composition, interleaving, caps), DB, HTTP + authz, clock-driven heartbeat tests.

**Migration strategy.** Additive; old lessons remain valid (`plan` NULL).

**Current status.** `NOT_STARTED`.

**Known limitations.** Application slice needs content (`communication_task` items) to be rich.

---

### P8 — Dashboard and Analytics

**Objective.** One honest, empty-state-aware dashboard: recommendations, review queue, weekly statistics, skill estimates, mastery stats, rule-based insights.

**Current implementation.** Program/weekly scales, lesson totals, exercise counts; no aggregation.

**Required changes.** Aggregation queries and read model; endpoints; insight rules; empty-state design; no-fake-stats test; performance indexes; optional rollup.

**Dependencies.** P2, P3, P6, P7; (D-012 for sharing).

**Database changes.** Indexes; optional `learner_daily_activity` only if measured necessary.

**API changes.** `GET /learning/dashboard`, `GET /learning/stats/weekly`.

**Implementation tasks.**
- P8-01 Aggregation queries (M) · P8-02 Dashboard endpoint (M) · P8-03 Review queue summary (S) · P8-04 Weekly stats with tz bucketing (M) · P8-05 Insight rules with evidence ids (M) · P8-06 Empty/incomplete states (S) · P8-07 No-fake-stats test (S) · P8-08 Daily recommendation rules (M) · P8-09 Query performance check with seeded volume (M) · P8-10 Rollup table if needed (M)

**Acceptance criteria.** Every field traces to a query (test enumerates fields); `null` + reason when below the minimum sample; empty/new/partial learner fixtures return valid payloads; p95 query time within budget on a seeded 100k-evidence profile (budget agreed at start); recommendations deterministic.

**Testing requirements.** DB aggregation tests with fixtures, golden JSON, HTTP + authz, performance test.

**Migration strategy.** Read-only; no data migration.

**Current status.** `NOT_STARTED`.

**Known limitations.** Targets for product measures (`PRODUCT_VISION.md` section 6) are set only after baselines exist.

---

### P9 — AI Tutor Integration

**Objective.** The tutor knows the learner (profile, recent mistakes, relevant knowledge items), holds practical scenarios, and produces *validated* learning evidence feeding the mistake journal and review system.

**Current implementation.** Four chat kinds with teacher mode, `✏️` free-text corrections, scenarios, cheap-model summaries, token budgets; DeepSeek-only streaming.

**Required changes.** Context builder (stable per dialog to keep the provider prefix cache); KB retrieval; scenario-to-task link; evidence extraction contract C-6 with audit table; mistake journal link; other-provider chat profiles (D-023).

**Dependencies.** P2, P5, P6, **D-023**.

**Database changes.** `ai_extractions` audit table; `learning_chats.context_snapshot` (jsonb) if needed.

**API changes.** Chat `done` event gains optional `evidence` summary; no breaking change.

**Implementation tasks.**
- P9-01 Learner-context builder (profile, top gaps, recurring errors, goal) with size cap (M) · P9-02 KB retrieval for the current message/topic (relations, tags, simple lexical match) (M) · P9-03 Context snapshot per dialog preserving append-only prompt (M) · P9-04 Scenario <-> `communication_task` binding (M) · P9-05 Evidence extraction + validators + audit (L) · P9-06 Journal/review integration (M) · P9-07 Provider chat profiles (L) · P9-08 Tests/docs (M)

**Acceptance criteria.** Validation rejects hallucinated item ids, out-of-taxonomy categories and quotes not present in the message; accepted tutor evidence has weight <= 0.5 and can never alone make an item `mastered`; chat prompt prefix stays stable across turns (existing prefix-stability test pattern extended); budgets/limits unchanged.

**Testing requirements.** AI contract tests, validator tests, prefix-stability tests, HTTP + authz, fake-provider conversations.

**Migration strategy.** Additive; feature flag for evidence extraction (default off).

**Current status.** `NOT_STARTED`.

**Known limitations.** Quality of extraction unmeasured at launch; DeepSeek-only unless D-023 expands.

---

### P10 — Advanced Learning

**Objective.** Listening, speaking, pronunciation assessment, vocabulary retention, writing evaluation, extended communication tasks, advanced analytics, additional languages. *Outline level: each item is detailed (all twelve fields) when the phase is scheduled and after the P8 baselines exist.*

**Current implementation.** TTS with cache, browser speech recognition (similarity >= 0.8), OCR, `write`/`read` rubric-less AI judging, tutor scenarios.

**Required changes (outline).** Item-level listening tasks and assessment; server-side speech evaluation (needs audio upload decision); pronunciation items and rubric; lexeme SRS (D-010/K-1); writing rubric with human-agreement measurement; multi-turn tasks with outcomes; analytics extensions; new language onboarding runbook (D-021).

**Dependencies.** P1-P9; D-010, D-021, D-015; storage/privacy decision for audio.

**Database/API changes.** Designed at phase start (`API_CONTRACTS.md` section 5, last row).

**Implementation tasks.** P10-01 Listening items and assessment · P10-02 Speaking evaluation pipeline · P10-03 Pronunciation items/rubric · P10-04 Lexeme retention · P10-05 Writing rubric calibration · P10-06 Extended tasks · P10-07 Advanced analytics · P10-08 Language onboarding runbook.

**Acceptance criteria / testing / migration.** Defined per task at phase start under the same `VERIFIED` rules.

**Current status.** `NOT_STARTED`.

**Known limitations.** Audio privacy, cost and browser support unresolved.

---

## 5. Prioritised backlog and critical path

Order = value first, risk second; items marked `||` can run in parallel with the previous one.

| Rank | Task | Why now | Size | Needs |
| --- | --- | --- | --- | --- |
| 1 | Owner answers D-001..D-005, D-020 (P0-10) | unblocks everything | — | owner |
| 2 | P0-11 Frontend enum tolerance | removes the main rollout hazard (R-07); tiny; independent | S-M | D-002 |
| 3 | P0-05 Re-runnable DB tests | stops false alarms before any real work | S | D-002 |
| 4 | P0-09 Production shape report | de-risks every backfill | S | D-020 |
| 5 | P0-06/07 Contract + migration test harnesses | the safety net for P1+ | M+M | D-019 |
| 6 | P1 Knowledge Base (P1-01..P1-11) | the foundation of identity; zero visible change | XL | D-007..D-010, D-014, D-018 |
| 7 | P2 Learner Profile + evidence dual-write | start accumulating evidence as early as possible (it cannot be recovered retroactively beyond what is stored) | L | P1, D-011 |
| 8 || P3-01/02 pure assessment engine + simulation | no dependencies on I/O; de-risks the hardest logic | L | none (pure) |
| 9 | P4-01 prerequisite authoring (content stream) | long lead time; start early | XL | D-013 |
| 10 | P3 rest: shadow mode, switch | needs P1/P2 | XL | D-017, D-022 |
| 11 | P5 Exercise engine | evidence quality | XL | FE change |
| 12 | P4 builder/adjust | needs P3 gaps + prereqs | XL | |
| 13 | P6 Adaptive | needs evidence volume | XL | D-010, D-016 |
| 14 | P7 Sessions | | L | |
| 15 | P8 Dashboard | needs data in P2-P7 | L | |
| 16 | P9 Tutor | | XL | D-023 |
| 17 | P10 | | XL | |

**Critical path:** P0-10 -> P1 -> P2 -> P3 -> P4 -> P6 -> P7 -> P8. **Off-path accelerators:** P3 pure engine (can start after P0), prerequisite content (start during P1), frontend tolerance and harnesses (before P1).

**Quick wins independent of the plan:** P0-05, P0-11, retiring/rewriting `CODING_STYLE.md` (D-004), adding `AGENTS.md` (D-005).

> **Pending owner decision D-024.** `BACKEND_BRIEF_FIT.md` section 5 proposes cutting the first
> steps into a thin end-to-end loop (S0-S4) before the layered order above, and section 4 lists
> eleven small additions (A-1..A-11) to the tasks of P1-P8 and a new cross-cutting task X-1 (AI
> gateway hardening). Until the owner answers, this plan is unchanged.

## 6. Risks (also see `CURRENT_STATE.md` section 11)

| ID | Risk | Mitigation in this plan |
| --- | --- | --- |
| R-01 | prod data unseen | P0-09 report; backfill count checks; restorable backup before each deploy |
| R-02 | push = deploy, no staging | small steps, flags, migration harness, shadow modes |
| R-03 | content effort (prereqs, C1/C2, new pairs) | separate content stream; design generic; D-013 |
| R-04 | BYO-key cost | deterministic-first, call budgets per contract, metering by purpose |
| R-05 | provider variance | validators + one retry + golden tests; provider-neutral contracts |
| R-07 | closed frontend enums | P0-11 first; flags/columns instead of new enum values |
| R-08 | thresholds untuned | simulation now, calibration in P8, no user-facing claims before |
| R-09 | scope creep across phases | one phase in progress; `VERIFIED` definition; status file is the gate |
