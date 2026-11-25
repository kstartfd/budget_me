# Current state of Learnings (audit)

Audit date: **2026-11-25**. Code audited: `main` at commit **`4662ad7`** (clean working tree).
Method: read the code, migrations, SQL queries, prompts, CI workflow and existing docs;
ran the backend test suite against real Postgres and the frontend gates (results in
section 9). Production was **not** accessed: production data volumes are unknown
(see R-01).

Statements are facts from the code unless marked `(unverified)`.

---

## 1. What this application is

A **Telegram Mini App** (also usable in a browser) for learning languages, with a Chrome
extension integration. One repository (`budget_me`) holds two independent apps that share
one account system:

| Part | Path | Notes |
| --- | --- | --- |
| Learnings backend | `apps/learning/backend` | Go, Echo, pgx, sqlc, goose |
| Learnings frontend | `apps/learning/frontend` | Next.js 16, React 19, shadcn/ui (Radix), Tailwind 4, next-intl |
| Shared account | `libs/account` | login / profile / deletion, own database `account_app` |
| Budget app | `apps/budget` | unrelated to this plan; do not touch |
| Infra | `infra/learning`, `infra/edge`, `infra/budget` | Compose, Caddy, blue-green deploy |

**Business model constraint that shapes everything:** AI is *bring your own key*. Every
learner stores their own provider key (Anthropic, OpenAI, DeepSeek, Kimi); the key is
sealed with AES-256-GCM (`internal/secretbox`, derived from `AUTH_SECRET`, bound to the user
row). Every AI feature spends **the learner's** key, so cost, rate limits and "no agent
configured" are first-class states in the product.

## 2. Technology stack

| Layer | Technology |
| --- | --- |
| Language / runtime | Go 1.25 (`go.mod`), module `learning_app/backend` |
| HTTP | Echo v4; envelope `{"code":0,"message":"success","data":...}` |
| DB access | pgx v5 + **sqlc** (`queries/*.sql` -> `internal/db`), no ORM; sqlc is the repository layer |
| Migrations | goose, embedded (`migrations/`, `cmd/migrate`), 35 files |
| Database | PostgreSQL 16; **two** databases: the app DB and the shared `account_app` DB |
| AI | own thin client `internal/llm` (Anthropic SDK + raw HTTP for OpenAI/DeepSeek/Kimi) |
| OCR | `tesseract` binary on the server (`internal/ocr`) |
| TTS | Google-backed provider (`TTS_URL`) + on-disk cache (`internal/learningaudio`) |
| Frontend | Next.js 16 (webpack build), React 19, TanStack Query, react-hook-form + zod, shadcn/ui + Radix, Tailwind 4, next-intl (ru/en/es), Recharts, Playwright |
| CI/CD | GitHub Actions: tests -> GHCR images -> SSH blue-green deploy; **push to `main` = production deploy** |
| Hosting | one VPS shared with the budget app, one Caddy, one Postgres |

Do **not** replace this stack without explicit approval (plan section 2).

## 3. Repository map (learning backend)

`apps/learning/backend/internal/` — **package-by-feature**, controllers call command/query
handlers directly (no bus). 23,300 lines of non-test Go, 10,819 lines of tests (50 test files).

| Package | Responsibility |
| --- | --- |
| `learning` | Notebook: topics (by language), entries (words/phrases), capture queue for the extension, review (SRS), daily sets, packs, fill-with-AI, entry audio |
| `learningpath` | **The teacher/program/test/exercise domain** (the part this plan evolves): teachers, placement & final tests, program build, exercises, attempts, lessons, rule book, rule practice |
| `learningpath/curriculum` | Embedded JSON library (the de-facto knowledge base) |
| `learningpath/placement` | CEFR ladder types, grading helpers, string similarity |
| `learningchat` | Chats: entry chat, free teacher chat (+ scenarios), path-topic chat, rule-example chat, dialogs archive, budgets |
| `learningai` | AI settings, `POST /ai/complete`, `POST /ai/scan` |
| `learningmarks` | Learner's highlight "triggers" for chat text |
| `learningtokens` | Extension tokens (capture / read_write) |
| `learningusage` | Records per-hour token usage of every AI call (`learning_agent_usage`) |
| `learningaudio` | TTS with cache |
| `llm` | Providers, model catalog, chat profiles, usage recorder |
| `ocr`, `ratelimit`, `secretbox`, `remind`, `tgbot` | infrastructure helpers (picture reader, token bucket, key sealing, reminders, Telegram bot) |
| `auth`, `account`, `accountlink`, `token` | sessions, shared-account glue |
| `httpx`, `apperr`, `db`, `ids`, `clock`, `config`, `i18n`, `log` | cross-cutting |

Entry points (`cmd/`): `http_server` (also runs the capture-queue consumer and the
account-deletion sweep), `migrate`, `remind` (cron).

Frontend (`apps/learning/frontend/src`): `app/learning/{page,[id],path,path/rules,path/rules/[id],chat,settings,tokens}`
routes; features `learning`, `path`, `chat`, `ai`, `scan`, `account`, `auth`, `telegram`;
22 shadcn primitives in `components/ui`; strings in `messages/{ru,en,es}.json` (768 lines each).

## 4. Glossary — two meanings of "topic"

This is the biggest naming trap for a newcomer. **Two separate content universes exist and
are not linked in the database:**

| Universe | Tables | Meaning |
| --- | --- | --- |
| **Notebook** | `learning_topics` -> `learning_entries` (+ captures, chats, scans, topic plans) | A personal word/phrase notebook per language. `learning_topics` = a *notebook* (e.g. "English"). Entries are cards with SRS. |
| **Teacher / program** | `learning_teachers`, `learning_paths` -> `learning_path_topics` -> `learning_path_exercises` -> `learning_path_attempts` (+ tests, jobs, lessons) | A structured course. `learning_path_topics` = a *lesson unit* of a program (a rule or theme). |

The only bridges: a "save to notebook" action in the UI, chats (`learning_chats` points to
either an entry, a notebook topic, a path topic or a rule-practice sitting) and the shared
language codes. The new platform must decide how these two universes relate (see
`DECISIONS.md` D-010).

## 5. Feature inventory (what already exists)

Plan terminology in *italics* maps to the target domain (see `IMPLEMENTATION_PLAN.md` section 2).

### 5.1 Teachers and levels (*Learner Profile*, partly)
- A **teacher** = one studied language with the language its rules are explained in
  (`learning_teachers`, unique per `(user, language)`); optional `explain_in_target`
  (explain in the studied language). `PATCH /path/teachers/:id` switches it; unstarted
  exercise sets are dropped and rewritten in the new language.
- Languages are free BCP-47 codes; the **library** only exists for pairs listed in section 6.
- Learner preferences (interests, variant, goal, hours per week) are collected by a survey
  when a program is built and stored **per program** (`learning_paths.preferences`).
  `PUT /path/goal` changes hours/week of the running program.

### 5.2 Placement and final tests (*Assessment Engine*)
- **Placement test** climbs a CEFR ladder A1 -> C2. The learner's own agent *writes* each
  batch (8 questions per level; final test 10), backend grades each answer
  deterministically (case/space-insensitive exact option match), and after each full batch
  the **agent acts as "examiner"** and returns `{verdict: climb|stop, level}`. The backend
  only enforces guard rails: climb needs >= 50% correct, a perfect batch cannot stop,
  a stop places at the batch level or the one below (`validateVerdict`).
- Questions, keys and answers are persisted (`learning_path_test_questions`,
  `learning_path_test_answers`); question keys are **not** sent to the client.
- One open test per `(user, language, kind)`; refresh resumes it.
- **Final test** confirms the running program's level; a pass archives the program as
  `finished` and unlocks a placement that starts one rung higher.
- **Choose reached level**: after a reset the learner may pick any level up to the highest
  closed program's level without a test (`POST /path/levels`, program `source = chosen`).
- Questions only cover `grammar`, `vocabulary`, `reading`; listening/speaking are not assessed.

### 5.3 Program (*Learning Path Engine*)
- Built from a finished placement by a background job (`learning_path_jobs`): the agent
  **picks 8-14 library topics** (ids validated against the library, each once, >=2
  non-grammar, listening+speaking present when the library has them). The text of every
  topic (rule, words, goal) comes from the library, written by people.
- Pairs/levels **without a library** (any other language pair, or reached level above B2):
  the agent writes the whole program (`purpose=program`), the case the curriculum code itself
  calls risky ("an agent that invents a grammar rule can invent a wrong one").
- After saving, exercise sets for the first `PathPrefetch` topics are written in the
  background, 3 at a time (budget 25 min).
- One active program per `(user, language)` (partial unique index). `POST /path/reset`
  archives it into history; `DELETE /path/history?language=` **hard-deletes** closed programs
  with everything under them.
- The program is **static after creation**: no re-ordering, no gap-driven insertion, no review
  items.

### 5.4 Exercises and attempts (*Exercise Engine*)
- Nine kinds: `choice`, `fill`, `translate`, `write`, `listen`, `speak`, `order`, `cloze`, `read`
  (`learning_path_exercise_kind` enum).
- Sets are written by the agent per topic and **persisted** (`learning_path_exercises`),
  versioned by `generation` (`learning_path_exercise_versions`); regenerating keeps old
  attempts. Structural validation exists per kind (e.g. 4 options with exactly one match,
  order words shuffled, cloze gap numbering, passage length).
- Grading: `choice`, `fill`, `listen`, `order`, `cloze` deterministically against the key
  (normalised text); `speak` by string similarity >= 0.8 against browser speech recognition;
  `translate`, `write`, `read` by the agent. On a wrong answer the agent writes an explanation
  in the learner's explanation language; the attempt stores `answer, correct, feedback, mistakes`.
- **Topic completion** = every exercise of the *latest* generation has **at least one correct
  attempt ever** (`CompleteLearningPathTopicIfExercisesPassed`). This is completion, not mastery.
- Exercise responses **include the answer key** (`exerciseResponse.answer`). This is a deliberate product choice ("guided exercise": the UI shows the key and the rule after a miss or a give-up, `features/path/schemas.ts`), but it means the browser holds the key **before the first attempt**, so a correct answer is not reliable evidence of recall (G-07). Placement-test questions, by contrast, never carry the key.

### 5.5 Lessons (*Study Session Engine*, partly)
- A **lesson** is a server-clocked 30-minute sitting (`LessonLength` constant) owned by the
  learner (not a topic since migration 0025), with pause/resume, expiring lazily. Every
  attempt made while it runs is tagged with `lesson_id`. It does **not** generate content;
  the learner chooses topics. Totals feed a weekly scale against `hoursPerWeek`.

### 5.6 Rule book and rule practice (*Knowledge Base*, partly)
- The rule book = library topics that have an `Extra` (forms, examples, mistakes, related
  rules): 45 rules (English pairs), 51 (Spanish pairs). Written by people, never by AI.
- **Rule practice** is a transient *sitting* per `(user, rule)`: the agent adds example
  sentences and 4 drills (`form`, `choose`, `correct`, `make`); lines are graded against the
  key without an agent; a chat per example. It is deleted 10 minutes after the learner
  leaves the page (6 h otherwise) — **by design no history is kept** (user decision,
  2026-11-12), so it produces no learning evidence.

### 5.7 Notebook and spaced repetition (*Adaptive Learning*, partly)
- Cards (`learning_entries`): text, transcription, translation, explanation, native usage,
  example, rule, source, status `inbox|learning|known`, text hash for dedup.
- **Leitner SRS** on cards only: stages with rest days `1,2,4,7,15,30,60`; grades
  `again` (back to stage 0, due in 10 min), `good` (+1), `easy` (+2); past the last stage
  the card becomes `known` (`NextReview`).
- **Daily sets** per notebook topic: learner chooses `daily_new` (1-50); missed days add
  to a capped credit (100); weekly/monthly re-reviews; a `weak` flag set when a card is
  graded "not known" in a set.
- Capture queue for the extension (Postgres `FOR UPDATE SKIP LOCKED`, idempotency keys,
  retries), AI card enrichment (`enrich`, `fill`), picture OCR import, AI "packs"
  (20 phrases / 10 words), Telegram reminders.

### 5.8 Chats (*AI Tutor*, partly)
- Four chat subjects in one table (`learning_chats`, constraint "exactly one subject"):
  notebook entry, notebook topic (free **teacher** chat with optional scenarios:
  restaurant, travel, shopping, interview), path topic, rule example.
- Teacher mode corrects with a `✏️` section; the correction is **free text only**.
- Streaming chat exists for **DeepSeek only** (`chatProfiles` in `llm/chat.go`); other
  providers answer 400.
- Cost controls: per-chat 120k tokens / 24h window, per-user 400k and per-key 800k tokens
  per UTC day, token bucket 5 + 1/4 s per user, history summarised by a cheap model.

### 5.9 AI infrastructure
- `llm.Client.Complete(ctx, Request) -> string` (text in, text out), providers
  Anthropic / OpenAI / DeepSeek / Kimi with per-model effort levels in `llm/catalog.go`.
- Path-domain calls go through `Agent.completeJSON`: read the learner's sealed key, call,
  parse JSON, run a validator; on a complaint **retry once** with the complaint appended.
- Every call is metered into `learning_agent_usage` (per user, hour, provider, model,
  purpose: input / cached / output tokens). Money is not stored.
- Full call inventory: `AI_CONTRACTS.md`.

### 5.10 Auth and authorization
- Session token (`v1.…`, stateless HMAC, 30 days) from Telegram initData or email+password
  via `libs/account`. Extension tokens (`bml_…`, SHA-256 hashed + sealed copy) with scope
  `capture` or `read_write`, optionally bound to one notebook topic.
- Authorization is **ownership scoping in SQL**: every query filters by `user_id`; foreign
  ids return 404. There are **no roles** (no admin, no teacher-viewer). Roles are planned
  separately (`admin-roles` plan, not started) — see `DECISIONS.md` D-012.

## 6. The knowledge library today

Embedded in the binary: `internal/learningpath/curriculum/data/<learned>_<explained>_<level>.json`,
`<learned>_<explained>_rules.json`.

| Pair (learned_explained) | Levels | Topics per level | Topics total | Rules | Key words |
| --- | --- | --- | --- | --- | --- |
| `en_ru` | A1-B2 | 20 / 21 / 22 / 22 | 85 | 45 | ~60 per level |
| `en_en` | A1-B2 | 20 / 21 / 22 / 22 | 85 | 45 | none |
| `es_ru` | A1-B2 | 23 / 23 / 23 / 22 | 91 | 51 | ~60 per level |
| `es_en` | A1-B2 | 23 / 23 / 23 / 22 | 91 | 51 | ~60 per level |
| `es_es` | A1-B2 | 23 / 23 / 23 / 22 | 91 | 51 | none |

Each level has the same skill mix: ~10-13 grammar, 3 vocabulary, 2 reading, 1 writing,
2 listening, 2 speaking topics. Topic fields: `id` (e.g. `a1-to-be`), `skill`, `unit`,
`title`, `goal`, `theory`, `vocabulary[]`, `tags[]`. Rules add `forms`, `examples`,
`mistakes`, `related`. Identifiers are unique per file and mirrored across explanation
languages (tests enforce the mirroring). **No C1/C2 content; no prerequisites; no
versioning; no sources.** Details and the target model: `KNOWLEDGE_BASE.md`.

## 7. Frontend state

- Screens: notebook (topics list, topic screen with flip-cards, review modal, daily-sets
  modal), entry detail with chat, **path screen** (teachers, level scale, program view,
  test runner, topic runner, lesson bar, weekly/program scales, rule book, rule practice),
  teacher chat screen, settings (AI agent, reminders, triggers, account), extension tokens.
- Design system: **shadcn/ui** over Radix with Tailwind 4; light/dark; palette variables in
  `app/globals.css`. i18n via next-intl in three locales kept in lock-step (768 lines each).
- Data layer: TanStack Query hooks per feature (`features/*/{api,hooks,schemas}.ts`), zod
  schemas for responses.
- The plan says frontend design references arrive separately; this plan changes **no UI**
  in Phase 0-1 and only adds screens as phases require.

## 8. Deployment and operations (summary)

GitHub Actions `ci.yml`: paths filter -> `test-learning-backend` (two Postgres 16 services,
`migrate up`, `go vet`, `go test`, tesseract installed), `test-learning-frontend` (`pnpm
lint`, `pnpm typecheck`) -> images to GHCR -> `deploy.yml` blue-green over SSH with a DB
backup before migration. **Every push to `main` that touches `apps/learning/**`,
`libs/account/**`, `infra/learning/**`, `infra/edge/**` or `.github/workflows/**` deploys to
production.** Plan consequence: every phase ships as small, backward-compatible steps
(expand -> migrate data -> contract). See `infra/budget/CI_CD.md`.

## 9. Baseline test results (Phase 0 task P0-04)

Run on 2026-11-25 against two fresh `postgres:16-alpine` containers (app DB + account DB),
Go 1.27.4 locally (CI uses 1.25), commit `4662ad7`:

| Gate | Command | Result |
| --- | --- | --- |
| Migrations | `go run ./cmd/migrate up` | OK, version 35 |
| Static analysis | `go vet ./...` | clean |
| Backend tests | `go test -count=1 -v ./...` | **262 passed, 0 failed, 2 skipped** (+13 subtests); all 17 packages with tests `ok` |
| Frontend lint | `eslint` | rc 0 |
| Frontend types | `tsc --noEmit` | rc 0 |
| Browser e2e | Playwright (6 tests) | **not run** (not part of CI; needs local stack, see `TESTING_STRATEGY.md`) |

The 2 skips are the real-tesseract tests (`tesseract` not installed on the audit machine; CI
installs it). Test counts by package: `learning` 75, `learningpath` 65, `learningpath/commands`
28, `llm` 22, `learningchat/commands` 13, `learning/commands` 10, `curriculum` 9, `ocr` 8,
`learningai/commands` 8, `auth` 7, `ratelimit` 4, `learningaudio` 4, `secretbox` 4,
`placement` 3, `remind` 2, `learningmarks` 1, `learningusage` 1.

**Known hazard (TD-03):** the DB tests are **not re-runnable on the same database on the
same UTC day**. A second `go test` against a database that already ran the suite fails 8
tests in `internal/learning` (`TestChat*`, HTTP 429 code `42903`) because the per-API-key
daily counter in `learning_chat_usage` (keyed by the tests' fixed fake key) is left at its
limit. Reproduced and confirmed: `TRUNCATE learning_chat_usage` makes the suite pass again.
Fresh DBs (CI) are unaffected. Rule: **truncate `learning_chat_usage` or use a fresh DB
before every local run.**

## 10. Gaps against the target (summary)

Full mapping and the backlog are in `IMPLEMENTATION_PLAN.md`. Condensed:

| ID | Gap | Impact |
| --- | --- | --- |
| G-01 | No canonical knowledge items: library is topic-level, JSON in the binary, ids are free text, no prerequisites/sources/versions; program topics reference it by an unconstrained `source_id` | cannot link evidence to items, cannot validate prerequisites, a library edit can orphan history |
| G-02 | Library only for 5 pairs, A1-B2; above B2 or other pairs the AI invents the whole program | content quality risk; contradicts "rules written by people" principle |
| G-03 | Pronunciation / functional language / communication objectives not modelled; vocabulary lists absent in `en_en`, `es_es` | |
| G-04 | Learner profile is thin: no per-language profile entity, no skill estimates, no item proficiency, no confidence | |
| G-05 | Placement verdict is made by the AI examiner (guard-railed); evidence is a boolean per AI-written question with no item link; result is one level string; no per-skill result, no confidence, no early/partial stop | violates "backend owns assessment evidence" |
| G-06 | Program is static; completion != mastery (one correct answer per exercise ever) | |
| G-07 | Attempts are not linked to knowledge items and lack attempt number, hints, latency, recall-vs-recognition, error category; exercise answer keys are sent to the client before the first attempt (by design for guided practice, so "show answer" is not recorded as a hint) | evidence is weak and forgeable |
| G-08 | Spaced repetition only for notebook cards; no unified review queue; no recurring-error detection | |
| G-09 | "Lesson" is a timer, not a generated balanced session; fixed 30 min | |
| G-10 | No dashboard aggregation; progress is scattered across endpoints | |
| G-11 | Tutor has no learner context or KB retrieval; corrections are free text and never extracted into a mistake journal | |
| G-12 | No roles (admin / teacher viewer) although authorization must be verified per phase | |

## 11. Technical debt and risks

| ID | Item |
| --- | --- |
| TD-01 | `apps/learning/backend/CODING_STYLE.md` describes a different architecture (hexagonal, `gin`, `wire`, MySQL, `IUserRepository`). Conflicts with the real code. |
| TD-02 | No agent instruction file in the repo; architecture rules live only outside it. `ARCHITECTURE.md` section 2 now records them. |
| TD-03 | Tests not re-runnable on the same DB the same day (section 9). |
| TD-04 | Frontend has no unit-test runner; 6 Playwright tests exist and are not in CI; one (`mobile details stay open…`) was recorded as failing for an unrelated reason in 2026-10 notes `(unverified now)`. CI frontend gate = eslint + tsc + image build. |
| TD-05 | Library edits require a deploy; content correctness is tested structurally only (`curriculum_test.go`), not linguistically. |
| TD-06 | Migration 0014's header says history is never deleted; `DELETE /path/history` hard-deletes (user-initiated). Conflicts with the plan's "never silently discard learning history". See D-011. |
| TD-07 | `learning_path_exercise_kind` is a Postgres enum extended with `ALTER TYPE … ADD VALUE` three times (0022, 0031); values can never be removed. |
| TD-08 | Two disjoint "topic" universes (section 4). |
| TD-09 | `learning_path_attempts` links only to an exercise (and optionally a lesson); there is no knowledge-item reference, attempt ordinal, hint flag, latency or error category. |
| TD-10 | Rate limiter is in process memory (documented); fine for one box. |
| R-01 | Production data volume/shape unknown to this audit; migrations must be rehearsed on a prod-shaped copy (needs an anonymised snapshot or row counts from the owner). |
| R-02 | Push to `main` = production deploy including migrations; no staging environment. |
| R-03 | Content scale: extending the library beyond 5 pairs needs human-reviewed content; AI-authored knowledge conflicts with the current "rules written by people" principle. |
| R-04 | BYO-key cost: adaptive features that call the AI more often spend the learner's money; budgets/pacing exist and must be extended, not bypassed. |
| R-05 | Provider variance in structured output quality; today mitigated by validators + one retry. |
| R-06 | Extension and capture API are external contracts (tokens in users' browsers); the notebook API must stay stable. |
| R-07 | The frontend parses status/kind fields with **closed `z.enum`s** (`features/path/schemas.ts`). A new enum value from the backend (e.g. test status `incomplete`, topic status `blocked`) breaks the existing screens until the frontend tolerates it; rollouts need frontend-first sequencing (`API_CONTRACTS.md` rule 3). |

## 12. Reusable components (do not rebuild)

| Need in the plan | Reuse |
| --- | --- |
| Provider-independent AI | `internal/llm` (+ `Agent.completeJSON`, usage recorder, purposes) |
| Structured AI output validation | validators in `learningpath/commands` (`validateBatch`, `validateVerdict`, `validateCatalogPlan`, `validateExercises`) |
| Deterministic grading | `placement.Grade`, `gradeOrder`, `gradeGaps`, `placement.Similarity`, practice key grading |
| Idempotent background work | capture queue pattern, `learning_path_jobs` |
| SRS | `NextReview` (Leitner) — fine for a first unified scheduler, see `LEARNING_ENGINE.md` |
| Time / sessions | `learning_path_lessons` (server clock, pause/resume) |
| Token & cost accounting | `learning_agent_usage`, chat budgets |
| Library loading + tests | `curriculum` package and `curriculum_test.go` structural tests |
| Account / authz primitives | `libs/account`, `httpx.RequireAuth`, SQL ownership scoping |
