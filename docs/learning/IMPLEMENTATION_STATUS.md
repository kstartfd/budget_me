# Implementation status

**The live record. Update this file in the same commit as every change to a task.**
Statuses: `NOT_STARTED` · `IN_PROGRESS` · `BLOCKED` · `IMPLEMENTED` · `VERIFIED`.
`VERIFIED` requires a row in section 3 (date, commit, command, result) — see
`TESTING_STRATEGY.md` section 7. Plan and acceptance criteria: `IMPLEMENTATION_PLAN.md`.

Last updated: **2026-11-25** · baseline commit `4662ad7` · active: **V1 (first vertical slice)**; P0 hygiene paused

## 1. Phase overview

| Phase | Name | Status | Blocked by | Notes |
| --- | --- | --- | --- | --- |
| V1 | First vertical slice (D-024) | **VERIFIED in production (smoke)** | — | tasks V1-02..V1-10 verified (V-002 local gate, V-003 CI and deploy) |
| P0 | Repository audit and baseline | **IMPLEMENTED** (docs), hygiene open | D-004, D-005, D-020 | docs written; baseline verified (V-001); P0-05..P0-11 not started |
| P1 | Knowledge Base | NOT_STARTED | D-002, D-003, D-008, D-009, D-010, D-014, D-018, D-020 | |
| P2 | Learner Profile | NOT_STARTED | P1, D-007, D-011 | |
| P3 | Assessment Engine | NOT_STARTED | P1, P2, D-017, D-022; pure engine (P3-01/02) can start earlier | |
| P4 | Learning Path Engine | NOT_STARTED | P1-P3, D-013, D-022 | |
| P5 | Exercise Engine | NOT_STARTED | P1, P2 | needs a frontend change (hidden keys) |
| P6 | Adaptive Learning | NOT_STARTED | P2, P4, P5, D-010, D-016 | |
| P7 | Study Session Engine | NOT_STARTED | P4-P6 | |
| P8 | Dashboard and Analytics | NOT_STARTED | P2, P3, P6, P7 | D-012 for sharing |
| P9 | AI Tutor Integration | NOT_STARTED | P2, P5, P6, D-023 | |
| P10 | Advanced Learning | NOT_STARTED | P1-P9, D-021 | outline only |

## 2. Task board

### V1 — First vertical slice

| Task | Description | Status | Evidence / note |
| --- | --- | --- | --- |
| V1-01 | Decisions, plan and status in the repo | IMPLEMENTED | `DECISIONS.md` D-002/D-024/D-025/D-026/D-030..D-033, `VERTICAL_SLICE_V1.md` |
| V1-02 | Migration 0036 + up/down/up test | VERIFIED | V-002; `migrations/loop_migration_test.go` up(35)->36->down->up with legacy rows |
| V1-03 | sqlc queries, pinned regeneration | VERIFIED | V-002; sqlc 1.31.1, `queries/learning_loop.sql` |
| V1-04 | `learningprofile` pure rules + tests | VERIFIED | V-002; `internal/learningprofile` (table + invariant tests) |
| V1-05 | Evidence, state, schedule in the attempt transaction | VERIFIED | V-002; `commands/loop.go`, `attempt.go`; rollback test |
| V1-06 | Review attempts via `activityId` | VERIFIED | V-002; `activityId` on the attempt endpoint |
| V1-07 | Activities (practice, review) | VERIFIED | V-002; `commands/activity.go` |
| V1-08 | Minimal dashboard | VERIFIED | V-002; `GET /learning/path/dashboard` |
| V1-09 | HTTP loop, rollback and security tests | VERIFIED | V-002; `internal/learningpath/loop_test.go` (15 tests) |
| V1-10 | Docs, full gates, push, CI watched | VERIFIED | V-003; pushed as `45c8641`; CI run 45621205693 green, `deploy-learning` success; prod `/healthz` 200, both new routes 401 without a session |

### P0 — Audit and baseline

| Task | Description | Status | Evidence / note |
| --- | --- | --- | --- |
| P0-01 | Audit repository, stack, architecture, AI, tests | IMPLEMENTED | `CURRENT_STATE.md`, `ARCHITECTURE.md` |
| P0-02 | Record schema (0001-0035) and API contracts | IMPLEMENTED | `DATA_MODEL.md` A, `API_CONTRACTS.md` 1-2 |
| P0-03 | Map to target, gaps, backlog, decisions | IMPLEMENTED | `IMPLEMENTATION_PLAN.md`, `DECISIONS.md` |
| P0-04 | Run baseline gates and record them | **VERIFIED** | section 3, row V-001 |
| P0-05 | Make DB tests re-runnable | NOT_STARTED | TD-03; needs D-002 go |
| P0-06 | Frontend zod contract-test harness | NOT_STARTED | needs D-019 |
| P0-07 | Migration up/down/up test harness | NOT_STARTED | |
| P0-08 | Coverage report (non-gating) | NOT_STARTED | |
| P0-09 | Production shape SQL report | NOT_STARTED | owner runs it (D-020) |
| P0-10 | Owner answers D-001..D-005, D-020 | NOT_STARTED | owner |
| P0-11 | Frontend enum tolerance release | NOT_STARTED | R-07 |

### P1 — Knowledge Base

| Task | Description | Status |
| --- | --- | --- |
| P1-01 | Body JSON Schemas; generalise library field checks to all pairs | NOT_STARTED |
| P1-02 | Migration `kb_*` (expand) | NOT_STARTED |
| P1-03 | `learningkb` package + idempotent importer | NOT_STARTED |
| P1-04 | Import topics of all pairs (85/91) | NOT_STARTED |
| P1-05 | Relations, categories, `authored` source | NOT_STARTED |
| P1-06 | Read interface (embedded/DB) + `LEARNING_KB_READ` flag | NOT_STARTED |
| P1-07 | Parity test (embedded vs DB, all pairs) | NOT_STARTED |
| P1-08 | Shadow mode compare | NOT_STARTED |
| P1-09 | Backfill `learning_path_topics.kb_item_id` | NOT_STARTED |
| P1-10 | KB read endpoints + authz tests | NOT_STARTED |
| P1-11 | `cmd/kb-import` + deploy hook | NOT_STARTED |
| P1-12 | Docs | NOT_STARTED |
| P1-13 | Contract: remove embedded read path | NOT_STARTED |

### P2 — Learner Profile

| Task | Description | Status |
| --- | --- | --- |
| P2-01 | Migration: teacher columns + `UNIQUE(id,user_id)` | NOT_STARTED |
| P2-02 | Migration: evidence, skill estimates, item state | NOT_STARTED |
| P2-03 | `learningprofile`: `RecordEvidence`, `ApplyEvidence` | NOT_STARTED |
| P2-04 | Dual-write evidence from attempts and test answers | NOT_STARTED |
| P2-05 | Backfill by replay (answers, attempts) | NOT_STARTED |
| P2-06 | `rebuild-learner-state` command | NOT_STARTED |
| P2-07 | Profile endpoints, survey prefill | NOT_STARTED |
| P2-08 | Authorization matrix | NOT_STARTED |
| P2-09 | Docs | NOT_STARTED |

### P3 — Assessment Engine

| Task | Description | Status |
| --- | --- | --- |
| P3-01 | Pure engine: posterior, selection, stop rules | NOT_STARTED |
| P3-02 | Simulation harness | NOT_STARTED |
| P3-03 | Migration for tests/questions/results | NOT_STARTED |
| P3-04 | Item sources + `assess.items/v2` | NOT_STARTED |
| P3-05 | Per-answer evidence, per-skill results | NOT_STARTED |
| P3-06 | Shadow mode vs AI examiner | NOT_STARTED |
| P3-07 | Flag switch + legacy `levelReached` (D-022) | NOT_STARTED |
| P3-08 | Start point / skill / diagnostic purposes | NOT_STARTED |
| P3-09 | Stop / incomplete / resume | NOT_STARTED |
| P3-10 | Gap analysis feeding `Analyze` | NOT_STARTED |
| P3-11 | Item analysis and quarantine | NOT_STARTED |
| P3-12 | Reassessment trigger predicates | NOT_STARTED |
| P3-13 | Docs; removal plan for `test-judge` | NOT_STARTED |

### P4 — Learning Path Engine

| Task | Description | Status |
| --- | --- | --- |
| P4-01 | Prerequisite authoring/import (content) | NOT_STARTED |
| P4-02 | `ValidatePath` + graph checks | NOT_STARTED |
| P4-03 | `Priority` function | NOT_STARTED |
| P4-04 | Deterministic builder | NOT_STARTED |
| P4-05 | `path.shape/v1` AI shaping | NOT_STARTED |
| P4-06 | Migration: revisions, topic flags | NOT_STARTED |
| P4-07 | `AdjustPath` | NOT_STARTED |
| P4-08 | Goal-based tasks | NOT_STARTED |
| P4-09 | Review activities in path | NOT_STARTED |
| P4-10 | Switch `Analyze` behind flag | NOT_STARTED |
| P4-11 | Endpoints + docs | NOT_STARTED |

### P5 — Exercise Engine

| Task | Description | Status |
| --- | --- | --- |
| P5-01 | Exercise schema v2 + validators | NOT_STARTED |
| P5-02 | Item-aware generation + backfill | NOT_STARTED |
| P5-03 | Attempt -> evidence in one transaction | NOT_STARTED |
| P5-04 | `reveal` endpoint, hidden keys, FE change | NOT_STARTED |
| P5-05 | Rubric scoring (C-2) | NOT_STARTED |
| P5-06 | Error taxonomy, classification, journal | NOT_STARTED |
| P5-07 | Deterministic grading extensions | NOT_STARTED |
| P5-08 | Topic-less exercises for review | NOT_STARTED |
| P5-09 | Docs | NOT_STARTED |

### P6 — Adaptive Learning

| Task | Description | Status |
| --- | --- | --- |
| P6-01 | Item `NextReview` (pure) | NOT_STARTED |
| P6-02 | State machine | NOT_STARTED |
| P6-03 | State update in evidence transaction | NOT_STARTED |
| P6-04 | Review queue + endpoint | NOT_STARTED |
| P6-05 | Mistake journal, recurring errors | NOT_STARTED |
| P6-06 | `AdjustPath` triggers | NOT_STARTED |
| P6-07 | Simulation (forgetting curves) | NOT_STARTED |
| P6-08 | Rebuild-equivalence test | NOT_STARTED |
| P6-09 | Notebook relation (D-010) | NOT_STARTED |
| P6-10 | Docs | NOT_STARTED |

### P7 — Study Sessions

| Task | Description | Status |
| --- | --- | --- |
| P7-01 | Migration | NOT_STARTED |
| P7-02 | `PlanSession` | NOT_STARTED |
| P7-03 | Pre-generation + deterministic fallback | NOT_STARTED |
| P7-04 | Persist / resume | NOT_STARTED |
| P7-05 | Heartbeat active time | NOT_STARTED |
| P7-06 | Session-end hooks | NOT_STARTED |
| P7-07 | Endpoints + docs | NOT_STARTED |

### P8 — Dashboard and Analytics

| Task | Description | Status |
| --- | --- | --- |
| P8-01 | Aggregation queries | NOT_STARTED |
| P8-02 | Dashboard endpoint | NOT_STARTED |
| P8-03 | Review queue summary | NOT_STARTED |
| P8-04 | Weekly stats | NOT_STARTED |
| P8-05 | Insight rules | NOT_STARTED |
| P8-06 | Empty/incomplete states | NOT_STARTED |
| P8-07 | No-fake-stats test | NOT_STARTED |
| P8-08 | Daily recommendations | NOT_STARTED |
| P8-09 | Performance check | NOT_STARTED |
| P8-10 | Rollup table (if needed) | NOT_STARTED |

### P9 — AI Tutor

| Task | Description | Status |
| --- | --- | --- |
| P9-01 | Learner-context builder | NOT_STARTED |
| P9-02 | KB retrieval | NOT_STARTED |
| P9-03 | Context snapshot per dialog | NOT_STARTED |
| P9-04 | Scenario <-> task binding | NOT_STARTED |
| P9-05 | Evidence extraction + audit | NOT_STARTED |
| P9-06 | Journal / review integration | NOT_STARTED |
| P9-07 | Other-provider chat profiles | NOT_STARTED |
| P9-08 | Tests / docs | NOT_STARTED |

### P10 — Advanced Learning

| Task | Description | Status |
| --- | --- | --- |
| P10-01..08 | Listening, speaking, pronunciation, lexeme retention, writing rubric, extended tasks, analytics, language onboarding | NOT_STARTED |

## 3. Verification log

Every `VERIFIED` status points here. Add rows; never edit old ones.

| ID | Date | Commit | Scope | Command(s) | Result |
| --- | --- | --- | --- | --- | --- |
| V-001 | 2026-11-25 | `4662ad7` (clean tree) | P0-04 baseline | two fresh `postgres:16-alpine`; `go run ./cmd/migrate up`; `go vet ./...`; `go test -count=1 -v ./...`; frontend `./node_modules/.bin/eslint`, `./node_modules/.bin/tsc --noEmit` | migrate OK (v35); vet clean; **262 PASS / 0 FAIL / 2 SKIP** (+13 subtests; skips = tesseract tests, binary absent locally); eslint rc 0; tsc rc 0. Same suite re-run on the *same* DB fails 8 `internal/learning` chat tests (429/42903) until `TRUNCATE learning_chat_usage` — TD-03, reproduced and cleared |
| V-002 | 2026-11-25 | `34ebfb1` + the V1 change set (committed next) | V1-02..V1-09 | two fresh `postgres:16-alpine`; `go run ./cmd/migrate up`; `go vet ./...`; `go test -count=1 -v ./...` (backend only; the frontend is unchanged) | migrate OK (v36); vet clean; **295 PASS / 0 FAIL / 2 SKIP** (+13 subtests; baseline 262 / 0 / 2), 19 packages ok; new: 15 loop tests, 17 rule tests, 1 migration test. Migration up(35)->36->down->up on seeded legacy rows leaves every row untouched. Not run: Playwright, production |
| V-003 | 2026-11-25 | `45c8641` | V1-10 | GitHub Actions on `main`, then `curl` against production | all jobs green (`test-learning-backend`, `test-learning-frontend`, `build-learning`, `deploy-learning`); migration 0036 ran on deploy; `/healthz` 200; `GET /learning/path/activities` and `/dashboard` answer 401 without a session. Not checked: a logged-in session in production (no credentials here) |

## 4. Open issues and known limitations (running list)

| ID | Issue | Since |
| --- | --- | --- |
| TD-03 | DB tests not re-runnable same day on one DB | 2026-11-25 |
| TD-04 | Frontend: no unit runner; e2e outside CI | 2026-11-25 |
| R-01 | Production data shape unknown | 2026-11-25 |
| R-07 | Closed frontend enums block new enum values | 2026-11-25 |

## 5. Session log (newest first)

### 2026-11-25 — session 2: first vertical slice V1

- **Did:** recorded the owner's decisions (D-024, D-002, D-025, D-026; D-003 interpreted; D-030..D-033), wrote `VERTICAL_SLICE_V1.md`, implemented V1: migration `0036_learning_loop.sql`, pure rules package `learningprofile`, evidence + knowledge state + review schedule in the attempt transaction, typed `practice`/`review` activities, review attempts via `activityId`, minimal dashboard, tests, docs.
- **Changed in the repo:** backend of the learning app (migration, queries, `learningprofile`, `learningpath` commands/queries/controller, tests) and `docs/learning/*`, `apps/learning/backend/docs/learning-api.md`. Frontend untouched. No existing response changed shape.
- **Behaviour change visible to production:** every answer to a library topic now also writes evidence and moves an item's state (data only; nothing the learner sees changes yet); two new session-only endpoints exist.
- **Limitations carried:** `VERTICAL_SLICE_V1.md` section 9.
- **Next:** watch CI and the deploy; then, in the agreed order, P0-05 (re-runnable tests), P0-11 (frontend enum tolerance) and the frontend design for the dashboard; then P1/P2 rest per D-024.

### 2026-11-25 — session 1b: fit analysis of the backend brief

- **Did:** compared the second brief ("Backend Implementation") with the code and the plan; wrote `BACKEND_BRIEF_FIT.md`; added D-024..D-029 and restated D-002.
- **Changed in the repo:** `docs/learning/*` only.
- **Result:** 6 rows covered, 11 need small additions, 2 real conflicts (order of work, who writes code). New cross-cutting task X-1 (AI gateway hardening) is proposed, not scheduled.
- **Next:** owner answers D-024 and D-002.

### 2026-11-25 — session 1: audit and documentation (Phase 0)

- **Did:** read the repository (backend, migrations, SQL, prompts, curriculum, chats, CI, frontend structure); ran baseline gates; wrote the 16 documents in `docs/learning/`.
- **Changed in the repo:** `docs/learning/*` only. No code, no migration, no config, no workflow touched; no production effect (outside the CI path filter).
- **Discovered (worth knowing first):** (1) AI examiner decides placement (G-05); (2) library has no prerequisites and is topic-level (G-01); (3) two separate "topic" universes (`CURRENT_STATE.md` section 4); (4) closed frontend `z.enum`s make new enum values a rollout hazard (R-07); (5) exercise answer keys reach the browser before the first attempt (by design, G-07); (6) test re-run hazard (TD-03); (7) no `AGENTS.md`/`CLAUDE.md`, and `CODING_STYLE.md` is a stale generic template.
- **Not done / deliberately skipped:** any implementation; Playwright e2e; production inspection.
- **Next step for whoever continues:** get answers to `DECISIONS.md` section C (at minimum D-002, D-003, D-020), then P0-05 and P0-11, then P1.
