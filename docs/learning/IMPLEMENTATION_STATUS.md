# Implementation status

**The live record. Update this file in the same commit as every change to a task.**
Statuses: `NOT_STARTED` · `IN_PROGRESS` · `BLOCKED` · `IMPLEMENTED` · `VERIFIED`.
`VERIFIED` requires a row in section 3 (date, commit, command, result) — see
`TESTING_STRATEGY.md` section 7. Plan and acceptance criteria: `IMPLEMENTATION_PLAN.md`.

Last updated: **2026-11-27** · baseline commit `7e57245` + working tree · next: **vocabulary assessment foundations (P3-17)**, subject to P1 lexical content/identity; P0 hygiene paused

## 1. Phase overview

| Phase | Name | Status | Blocked by | Notes |
| --- | --- | --- | --- | --- |
| V1 | First vertical slice (D-024) | **VERIFIED in production (smoke)** | — | tasks V1-02..V1-10 verified (V-002 local gate, V-003 CI and deploy) |
| P0 | Repository audit and baseline | **IMPLEMENTED** (docs), hygiene open | D-004, D-020 | docs written; baseline verified (V-001); D-005 resolved; P0-05..P0-11 not started |
| P1 | Knowledge Base | NOT_STARTED | D-002, D-003, D-008, D-009, D-010, D-014, D-018, D-020 | |
| P2 | Learner Profile | NOT_STARTED | P1, D-007, D-011 | |
| P3 | Assessment Engine | IN_PROGRESS | Vocabulary/reading content and audio/rubric dependencies | Grammar-only [Quick Screening](QUICK_SCREENING.md) verified locally (V-004); D-035..D-037 expand target to three modes |
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
| P0-10 | Owner answers D-001..D-005, D-020 | IN_PROGRESS | D-005 accepted; other open answers remain |
| P0-11 | Frontend enum tolerance release | NOT_STARTED | R-07 |

### P1 — Knowledge Base

| Task | Description | Status |
| --- | --- | --- |
| P1-01 | Body JSON Schemas; generalise library field checks to all pairs | NOT_STARTED |
| P1-02 | Extend `learning_knowledge_items` and add other `learning_knowledge_*` tables (preserve `kb_items` view) | NOT_STARTED |
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

The 2026-11-26 scope update is specified in
[`ASSESSMENT_MODES.md`](ASSESSMENT_MODES.md). The existing `VERIFIED (Quick Screening)`
rows below refer **only to the grammar slice**. They do not mean the full Quick mode,
Full Assessment or Skill Assessment is available.

| Mode / skill capability | Status | Evidence / remaining work |
| --- | --- | --- |
| Quick / grammar (English, Spanish) | VERIFIED locally (V-015) | Shared section engine; onboarding to result/path, batch adaptation, evidence and resume |
| Quick / vocabulary | NOT_STARTED | Independent lexical blueprint, keys and evaluator |
| Quick / reading | NOT_STARTED | Passage-linked blueprint and comprehension evaluator |
| Quick / all three skills | NOT_STARTED | Multi-section orchestration and aggregate result |
| Skill Assessment | NOT_STARTED | Section selection, repeatable runs and history/comparison |
| Full Assessment | NOT_STARTED | Independent section storage exists; orchestration and required-skill completion do not |
| Writing | NOT_STARTED | Rubric and calibration |
| Listening | NOT_STARTED | Audio-linked item source and comprehension gate |
| Speaking | NOT_STARTED | Recording, upload, audio evaluation |
| Pronunciation | NOT_STARTED | Validated acoustic evaluation |

P3-14..P3-25 are the revised work items in `IMPLEMENTATION_PLAN.md`. P3-14 through
P3-16 are verified locally; no additional mode or skill is available.

| Revised task | Status | Evidence / remaining work |
| --- | --- | --- |
| P3-14 Shared mode/skill blueprint | VERIFIED locally (V-008) | Versioned runtime registry, lifecycle validation, source/writer/evaluator interfaces wired into Quick, pure aggregation, grammar section result and capability API. |
| P3-15 Independent section storage/results | VERIFIED locally (V-014) | Migration 0039, section policy/blueprint and result storage, question/answer/evidence attribution, old-writer bridge and guarded up/down/up. Independent multi-section orchestration remains P3-21. |
| P3-16 Grammar section through shared engine | VERIFIED locally (V-015) | Uses the persisted policy and blueprint for target allocation, validation, evaluation, adaptation and result aggregation. Browser regression covers the grammar Quick journey. |
| P3-17..P3-25 | NOT_STARTED | See the ordered dependencies in `IMPLEMENTATION_PLAN.md`. |

| Task | Description | Status | Evidence / note |
| --- | --- | --- | --- |
| P3-01 | Pure engine: posterior, selection, stop rules | VERIFIED (Quick Screening, V-004) | Heuristic policy implemented for Quick Screening. Posterior deferred. |
| P3-02 | Simulation harness | NOT_STARTED | |
| P3-03 | Migration for tests/questions/results | VERIFIED (Quick Screening, V-004) | Migration 0037 extends existing tests, questions, attempts and evidence. |
| P3-04 | Item sources + `assess.items/v2` | VERIFIED (Quick Screening, V-004) | Curated bank and strict fallback for Quick Screening. |
| P3-05 | Per-answer evidence, per-skill results | VERIFIED (Quick Screening, V-004) | Assessment attempt ledger and grammar estimates. Other skills stay unknown. |
| P3-06 | Shadow mode vs AI examiner | NOT_STARTED | |
| P3-07 | Flag switch + legacy `levelReached` (D-022) | NOT_STARTED | |
| P3-08 | Start point / skill / diagnostic purposes | VERIFIED (Quick Screening, V-004) | Self-reported start implemented. Additional purposes reserved. |
| P3-09 | Stop / incomplete / resume | VERIFIED (Quick Screening, V-004) | Bounded policy, partial results and durable resume implemented. |
| P3-10 | Gap analysis feeding `Analyze` | VERIFIED (Quick Screening, V-004) | Screening results feed the existing build job and canonical library. |
| P3-11 | Item analysis and quarantine | NOT_STARTED | |
| P3-12 | Reassessment trigger predicates | NOT_STARTED | |
| P3-13 | Docs; removal plan for `test-judge` | VERIFIED (Quick Screening, V-004) | Quick Screening methodology and API documented. Legacy examiner retained. |

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

### Overview frontend integration (working tree, 2026-11-26)

The existing Overview now uses the design's two-column layout for a learner with
a Teacher but no program. The three starting choices use the existing onboarding
state machine; all seven skills and the activity/plan cards show explicit empty
states. When every topic in an active program is completed, the Overview shows a
completed-program state and a route to additional practice. Due review cards can
open a real exercise and submit its `activityId` through the existing attempt API;
the dashboard query is refreshed after the answer. This does **not** implement a
daily study-session completion model, skill estimates, insights, or the full P8
analytics phase. See [Overview components](../../apps/learning/frontend/src/features/home/components/),
[attempt API client](../../apps/learning/frontend/src/features/path/api.ts), and
[browser test](../../apps/learning/frontend/e2e/learning.overview.anon.spec.ts).

The later local navigation pass adds the eight-section desktop sidebar and a
Telegram-width menu, routes for Knowledge Base, Practice and Progress, selected
language propagation, the archive-and-restart **Start a new plan** flow, and
honest boundaries for notebook/free practice, AI Tutor and missing skill
estimates. See [frontend handoff](frontend/NAVIGATION_HANDOFF.md). This is UI
integration over existing APIs, not completion of P1, P3, P6, P8 or P9.

Local gate on the working tree: frontend `pnpm build`, `pnpm typecheck`, targeted
ESLint and seven Chromium Playwright scenarios passed (five onboarding, two
Overview including new learner and due review). This is frontend integration
evidence only; no production verification
or P8 `VERIFIED` status is claimed.

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
| V-004 | 2026-11-25 | `6440236` + Quick Screening working tree | Quick Screening MVP | Fresh PostgreSQL 16 databases, `go run ./cmd/migrate up`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./internal/learningpath/... ./migrations`, frontend ESLint and TypeScript, Chromium `learning.assessment.auth.spec.ts` | Passed. Migration 0037 down/up preserved seeded legacy tests. Browser verified pause, reload, resume, skip, partial results and path creation. |
| V-005 | 2026-11-26 | `6440236` + working tree | P3-14 first code slice; Quick grammar regression | Two fresh disposable PostgreSQL 16 databases, migration to 0037, `go vet ./...`, `go test -count=1 ./...` | Passed. All backend packages green. New runtime blueprint guards unsupported modes and the Quick result exposes one grammar section. Frontend/browser was unchanged and not rerun; P3-14 and other modes remain in progress/not started. |
| V-006 | 2026-11-26 | `6440236` + working tree | P3-14 capability contract and regression | Fresh disposable PostgreSQL 16 app/account databases, migrate to 0037, `go vet ./...`, `go test -count=1 ./...`, frontend targeted ESLint and `tsc --noEmit` | Passed. New HTTP test checks en/es grammar-only capability, unsupported language and session auth; existing Quick end-to-end backend test checks section result. Containers removed. Browser test not rerun; Full/Skill/other skills remain unavailable. |
| V-007 | 2026-11-26 | `6440236` + working tree | P3-14 pure section orchestration wired into Quick | Fresh disposable PostgreSQL 16 app/account databases, migrate to 0037, `go vet ./...`, `go test -count=1 ./...` | Passed. Pure tests cover new/paused/completed/partial/malformed multi-section states; Quick completion now uses the shared aggregator. Containers removed. No Full/Skill workflow enabled. |
| V-008 | 2026-11-26 | `6440236` + working tree | P3-14 shared Assessment Engine contracts | Fresh disposable PostgreSQL 16 app/account databases, migrate to 0037, `go test -count=1 ./...`, `go vet ./...` | Passed. Lifecycle guard and question source/writer/evaluator interfaces are wired into Quick. Existing backend and migration tests pass. No frontend behavior changed in this slice; browser gate V-004 remains the last Quick journey check. Container removed. Full/Skill modes remain unavailable. |
| V-009 | 2026-11-26 | `6440236` + working tree | `kb_items` naming compatibility, migration 0038 | Fresh disposable PostgreSQL 16 app/account databases; migrate to 0038; `go test -count=1 -run TestKnowledgeItemTableRename -v ./migrations`; `go test -count=1 ./...`; `go vet ./...`; repeat the focused test on a second fresh PostgreSQL 16 after adding a pre-migration prepared statement check | Passed. Old `kb_items` SQL (including `ON CONFLICT ... RETURNING *` and a statement prepared before migration) and new `learning_knowledge_items` SQL share IDs; existing evidence FK and rows survive up/down/up. sqlc 1.31.1 regenerated learning backend only. No production deploy or frontend gate. |
| V-010 | 2026-11-26 | `6440236` + working tree | Eight-section navigation, Rule Book UI, Practice/Progress entry points, plan restart | `pnpm build`; `pnpm typecheck`; targeted ESLint; `E2E_CHROMIUM_PATH=/usr/bin/google-chrome pnpm exec playwright test e2e/learning.overview.anon.spec.ts e2e/learning.onboarding.anon.spec.ts e2e/learning.assessment.auth.spec.ts`; `git diff --check` | Build/typecheck/lint/diff clean. Chromium: **12 passed, 1 skipped** (authenticated assessment needs `E2E_SESSION_FILE`). Tested desktop/mobile navigation, selected language switch, real-schema Rule Book filters, Overview review and archive-to-starting-choices flow. Desktop and 390px mobile screenshots were inspected during local QA. No MagicPath publication or production deploy. |
| V-011 | 2026-11-26 | `6440236` + working tree | Earlier screenshot-based sidebar pass, superseded by V-012 | `pnpm typecheck`; targeted ESLint; `next build --webpack`; `E2E_CHROMIUM_PATH=/usr/bin/google-chrome pnpm exec playwright test e2e/learning.overview.anon.spec.ts`; side-by-side 231 × 1154px sidebar comparison; `git diff --check` | Gates passed, but the screenshot was the wrong design authority. Its visual acceptance claim is withdrawn. Footer controls moved to Settings without removal. No production deploy. |
| V-012 | 2026-11-26 | working tree | Sidebar aligned to the locally mirrored design component | `pnpm typecheck`; targeted ESLint; `E2E_CHROMIUM_PATH=/usr/bin/google-chrome pnpm exec playwright test e2e/learning.overview.anon.spec.ts`; visual inspection at 1440 × 900; `git diff --check` | Typecheck/lint/diff clean; Chromium **8 passed**, including the 900px breakpoint. Selector, navigation spacing/type, active row and footer inspected against the design source. Icons still use installed Tabler paths instead of prototype Lucide paths. No production deploy. |
| V-013 | 2026-11-26 | working tree | Local version-controlled design preview and screenshot handoff | Isolated copy without `node_modules`/`dist`; `npm ci --offline`; `npm run build`; `npm run dev`; `CHROME_PATH=/usr/bin/google-chrome npm run capture`; `cmp` of 20 generated PNGs | Install/build/start/capture passed. All **20 screenshots byte-identical** to the checked-in set. No backend or hosted design service used. No product feature marked implemented. |
| V-014 | 2026-11-27 | `7e57245` + working tree | P3-15 section storage | Fresh disposable PostgreSQL 16 app/account databases, `go run ./cmd/migrate up`, `go vet ./...`, `go test -count=1 ./...`; second fresh app/account pair, `go test -race -count=1 ./internal/learningpath/... ./migrations`; focused `go test -count=1 -run 'TestAssessmentSectionsMigrationAndOldWriter\|TestAssessmentSection\|TestQuickScreening' ./migrations ./internal/learningpath` | Passed locally. Migration 0039 up/down/up preserves legacy and new Quick snapshots, answers, outcomes and stable section IDs; tests cover an old transaction during migration, old prepared statements and writes after migration, partial/completed results, attribution, duplicate submissions, authorization, resume with changed server defaults, and rollback rejection for independent data. Two unrelated real-image OCR tests skipped: Tesseract was not installed. No UI change; browser not rerun. No production deployment or verification. |
| V-015 | 2026-11-27 | `7e57245` + working tree | P3-16 grammar section and Quick browser journey | Fresh isolated PostgreSQL 16 app/account databases: `go run ./cmd/migrate up`, `go vet ./...`, `go test -count=1 ./...`; fresh second pair: `go test -race -count=1 ./internal/learningpath/... ./migrations`; focused `go test -count=1 -run TestAssessmentSectionsMigrationAndOldWriter ./migrations`; frontend `pnpm lint`, `pnpm exec tsc --noEmit`, `pnpm build`; `E2E_SESSION_FILE=<isolated test session> E2E_BACKEND_URL=<local test server> E2E_FRONTEND_PORT=3302 E2E_CHROMIUM_PATH=/usr/bin/google-chrome E2E_SCREENSHOT_DIR=/tmp pnpm exec playwright test e2e/learning.assessment.auth.spec.ts --project=chromium-auth` | Passed locally: backend, migration/legacy HTTP and race gates; frontend lint, types and build; Chromium **3 passed**, at 1440 × 900 and 390 × 844 plus capability/auth checks. The browser covered English onboarding, question answer/retry, leaving/returning, pause, reload, resume, skip, partial result and path creation; Spanish Quick starts and German is unavailable. English/Spanish versus other languages, repeated answers, unauthenticated and cross-user access, and old attempts are covered by browser and/or backend tests. Two unrelated real-image OCR tests skipped because Tesseract was absent. The browser used the curated question bank; a live AI-provider generation call and a fully completed assessment were not exercised there. Visual comparisons are documented in [design QA](../../apps/learning/frontend/design-qa.md). No production deployment or verification; migration 0039 remains unverified in production. |

For V-015, `go` was `/snap/go/current/bin/go` and `pnpm` was
`/home/bist/.local/share/pnpm/bin/pnpm`; database and session values pointed only to
the disposable local test environment and are intentionally omitted.

## 4. Open issues and known limitations (running list)

| ID | Issue | Since |
| --- | --- | --- |
| TD-03 | DB tests not re-runnable same day on one DB | 2026-11-25 |
| TD-04 | Frontend: no unit runner; e2e outside CI | 2026-11-25 |
| R-01 | Production data shape unknown | 2026-11-25 |
| R-07 | Closed frontend enums block new enum values | 2026-11-25 |

## 5. Session log (newest first)

### 2026-11-27 — P3-16 grammar section execution

- Routed the released grammar Quick section through the shared `SectionEngine`,
  reading the persisted policy/blueprint and keeping legacy transactions, attempts,
  answers and API compatibility. The UI renders preliminary grammar evidence and
  does not offer the prototype-only next-section action.
- Verified locally as V-015, including the actual browser flow and a pre-0039
  attempt resumed through the migrated API. P3-17 follows, subject to P1 lexical
  content/identity. No production deployment or verification.
- The live, untracked preview has three Path creation states beyond the nine in the
  canonical frontend handoff; reconciliation is separate from this assessment slice.

### 2026-11-27 — P3-15 assessment section storage

- Added migration 0039, a compatibility bridge for old Quick writers, section
  policy/blueprint/result snapshots and an evidence attribution view. Results missing
  from older snapshots remain unknown; existing outcomes and evidence are preserved.
- Verified P3-15 locally as V-014. P3-14 was already locally verified in V-008;
  corrected the stale project-context description. P3-16 is next. Full, Skill and
  additional skill assessments are still unavailable. No production deployment.

### 2026-11-26 — Local Learning design handoff

- The [frontend design entry point](frontend/README.md) now owns a runnable
  local prototype, visual specification, screen/interaction map, and 20 desktop
  and Telegram-width captures. It imports the existing TSX source directly.
- The published MagicPath preview remains a historical, outdated revision; the
  local source is the latest intended design. Sample progress and future
  capabilities remain explicitly separate from verified app behavior.
- Gate V-013 verified a clean isolated install, build, start and exact screenshot
  regeneration. This documentation/design work changes no backend capability.

### 2026-11-26 — Learning frontend navigation and restart

- Local app now shows Overview, Learning Path, Knowledge Base, Practice,
  Assessments, Progress, AI Tutor and Settings in one desktop/mobile navigation.
  The Knowledge Base reads the existing Rule Book and labels the personal
  notebook separately. Practice opens real due reviews or program activities;
  Progress reads stored loop totals and shows no invented skill estimates.
- **Start a new plan** archives the active program through the existing reset
  API, then reopens the starting-method selection without deleting history.
  The separate history/progress deletion scope is still open.
- Frontend gate V-010 passed. The MagicPath external-agent quota (50/50)
  prevents publishing the revised design mirror; its existing preview must not
  be called the final prototype.

### 2026-11-26 — Sidebar design source correction

- The prior screenshot comparison used an outdated source and is not an
  acceptance criterion. The locally mirrored design component defines the
  sidebar: 232px frame, 12px navigation labels, 15px icons, 36px
  rows and a thin bordered active row. The local sidebar now follows those
  values while retaining live navigation and language state.
- Interface locale, theme and sign-out controls remain in Settings. The
  corrected visual comparison is recorded in
  `apps/learning/frontend/design-qa.md`; frontend gate V-012 passed.

### 2026-11-26 — Knowledge item table naming

- Migration 0038 renames the physical `kb_items` table to
  `learning_knowledge_items` and provides an updatable `kb_items` view for the
  old application's reads and writes during blue-green deployment. New sqlc
  queries use the physical name; existing item and evidence UUIDs are unchanged.
- Verified migration up/down/up, legacy and new upserts, foreign keys and the
  full backend gate on disposable PostgreSQL (V-009). No production deploy.
  P1 content/versioning/importer work remains unstarted.

### 2026-11-26 — Assessment scope expansion

- Completed P3-14 shared contracts (V-008): Quick uses a lifecycle transition
  guard, canonical question source, AI writer interface and keyed answer evaluator.
  No additional skills or modes were enabled. Independent section persistence and
  complete blueprint snapshots are the next P3-15 slice.
- Owner approved Quick (grammar, vocabulary, reading), Full (independent skill
  sections) and repeatable Skill Assessment through one engine; audio and rubric
  capabilities have explicit release gates (D-035..D-037).
- Added `ASSESSMENT_MODES.md`, reconciled the plan, decision log, architecture,
  data/API/AI contracts and status. The current grammar Quick slice remains separate
  from the target Quick scope.
- Started P3-14: reusable runtime mode/skill blueprint validation, a shared
  completion-status rule, a per-section grammar result and a read-only capability
  API. A pure multi-section aggregator now drives Quick's overall completion status.
  Existing onboarding and user data were not changed. Full backend gate V-007
  and earlier TypeScript check V-006 passed on disposable databases. The frontend preserves the
  additive section result but does not yet use the capability response for selection.

### 2026-11-26 — agent context handoff

- Created root `AGENTS.md` at the owner's request (D-005), `docs/PROJECT_CONTEXT.md`,
  and the ten-document `.context-export/` handoff package plus `learning-context.zip`.
- Reconciled the live documentation index, architecture introduction and D-005 status.
  No existing application source or user data changed in this documentation step.
- Owner's next product direction: align the sidebar with the original Overview design,
  map the tabs, and define an explicit start-from-zero flow before resetting a test user.

### 2026-11-25 — Quick Screening

- Implemented D-034 with the existing test and evidence stores, a canonical English
  and Spanish grammar question bank, bounded adaptation and an evidence-aware path build.
- Verified the local backend and browser flow (V-004). Broader screening modes, calibrated
  measurement and independent estimates for other language skills remain future work.

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
