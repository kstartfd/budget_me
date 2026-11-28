# Learnings project context

Snapshot: **2026-11-27**, branch `main` at `7e57245`, **including uncommitted work**.
This is a routing and status map for coding agents. For detailed design, follow the
[learning documentation index](learning/README.md); for exact current API payloads,
use the [canonical backend API reference](../apps/learning/backend/docs/learning-api.md).
The committed V1 loop was verified in production; the Quick Screening and new
frontend shown below are a working-tree state with a local verification record,
not a verified production release ([status](learning/IMPLEMENTATION_STATUS.md)).

**Assessment scope update, 2026-11-26:** the target product has Quick
(grammar/vocabulary/reading), Full (independent, resumable skill sections) and
repeatable Skill Assessment through one shared engine. Only the grammar Quick slice
is operational. The shared blueprint/lifecycle and capability API slice P3-14 was
verified locally in V-008. P3-15 adds persisted section storage and compatibility for
old Quick attempts. P3-16 routes grammar through the section engine and has a local
browser gate; see the [status ledger](learning/IMPLEMENTATION_STATUS.md) for the
production boundary. See
[current mode contract](learning/ASSESSMENT_MODES.md), D-035..D-037 and the
[revised roadmap](learning/IMPLEMENTATION_PLAN.md#p3--assessment-engine).

## 1. Technology stack

Learnings is a Telegram Mini App and web app. Its backend uses Go 1.25, Echo v4,
pgx v5, sqlc and goose on PostgreSQL 16. The frontend uses Next.js 16 App Router,
React 19, TypeScript, TanStack Query, zod, next-intl (ru/en/es), shadcn/ui, Radix,
Tailwind 4 and Playwright. The app and shared account system use separate databases.
AI is bring-your-own-key through an internal provider client (Anthropic, OpenAI,
DeepSeek, Kimi). GitHub Actions builds and deploys changed apps blue-green on a
shared VPS. Sources: [Go module](../apps/learning/backend/go.mod),
[frontend package](../apps/learning/frontend/package.json),
[deployment guide](../infra/learning/README.md).

## 2. Application architecture

One Go service, one Next.js app, one learning database and a separate shared-account
database form a modular monolith. Backend feature packages have thin Echo
controllers, direct command/query handlers, sqlc queries as the repository, and
transactions for multi-table writes. `internal/server/server.go` wires the packages;
there is no command bus or broker. The frontend uses route pages and feature folders
with API fetchers, zod response schemas and TanStack Query hooks. Ownership is
enforced in SQL by `user_id`; teacher/path/assessment/chat routes require a session.
See [actual conventions](learning/ARCHITECTURE.md#2-real-conventions-today-write-code-like-this),
[server wiring](../apps/learning/backend/internal/server/server.go) and
[app shell](../apps/learning/frontend/src/components/app-shell.tsx).

Two content universes still coexist: the personal notebook (`learning_topics`,
`learning_entries` and its SRS) and the structured teacher/program domain
(`learning_teachers`, paths, tests, exercises). Their topics are different entities.
The new knowledge/evidence loop applies to attributable program content; it does
not merge the notebook ([audit glossary](learning/CURRENT_STATE.md#4-glossary--two-meanings-of-topic),
[D-010](learning/DECISIONS.md#before-phase-1-starts)).

## 3. Repository structure

| Path | Responsibility |
| --- | --- |
| [`apps/learning/backend`](../apps/learning/backend) | Go API, `cmd/`, `internal/`, goose `migrations/`, sqlc `queries/`, API reference |
| [`apps/learning/frontend`](../apps/learning/frontend) | Next.js routes in `src/app`, feature modules in `src/features`, locale messages, Playwright e2e |
| [`libs/account`](../libs/account) | Shared identity, login, profile and deletion |
| [`apps/budget`](../apps/budget) | Separate application in the same repository |
| [`infra/learning`](../infra/learning), [`infra/edge`](../infra/edge) | Compose, Caddy and deployment integration |
| [`docs/learning`](learning) | Architecture, decisions, domain specifications, plan and live status |

## 4. Implemented functionality and current phase

| Subsystem | What runs now | What remains |
| --- | --- | --- |
| Teacher / Learner Profile | One teacher per learner and studied language, explanation language, active program; teacher is the intended profile root. The new wizard stores goals, skill priorities, routine and preferences in **browser localStorage**, while program preferences/hours are saved with the program. [Teacher commands](../apps/learning/backend/internal/learningpath/commands/teacher.go), [wizard profile](../apps/learning/frontend/src/features/home/profile.ts) | Server-owned goals, per-skill estimates/profile API and cross-device setup (P2) |
| Onboarding | New Overview, five-step language wizard, language management and three starting choices (`assess`, claimed level, A1 from scratch) exist in the working tree. [wizard](../apps/learning/frontend/src/features/home/components/wizard.tsx), [stage resolver](../apps/learning/frontend/src/features/home/setup.ts) | Reconcile all screens with the design and provide a deliberate full restart flow |
| Assessment Engine | Legacy AI-authored placement/final tests remain. Uncommitted Quick Screening adds a curated grammar bank, bounded deterministic band decisions, per-answer evidence, durable pause/resume/stop and partial results for English/Spanish. P3-15 persists a grammar section and supports old writers through migration 0039; P3-16 executes it through the shared section engine. [engine](../apps/learning/backend/internal/learningpath/assessment/engine.go), [commands](../apps/learning/backend/internal/learningpath/commands/assessment.go), [screen](../apps/learning/frontend/src/features/assessment/screen.tsx), [spec](learning/QUICK_SCREENING.md) | P3-17 vocabulary objectives and evaluator; further skills and modes |
| Knowledge Base / Rule Book | Human-authored embedded JSON library for five studied/explained pairs, A1-B2; Rule Book and transient free practice. V1 lazily creates knowledge identities from library topics. Migration 0038 names the physical table `learning_knowledge_items` and leaves a `kb_items` compatibility view. [curriculum](../apps/learning/backend/internal/learningpath/curriculum), [name migration](../apps/learning/backend/migrations/0038_learning_knowledge_items_name.sql), [KB specification](learning/KNOWLEDGE_BASE.md) | Full DB KB, versions, relations, prerequisites, importer and read switch (P1); tracked Rule Book practice |
| Learning Path | One active program per language; survey and a completed test/claimed level feed a background build. Supported library pairs select validated authored topics; Quick Screening additionally supplies gaps/checks to a deterministic planner in the working tree. [build](../apps/learning/backend/internal/learningpath/commands/build.go), [screening plan](../apps/learning/backend/internal/learningpath/commands/assessment_plan.go) | Prerequisite graph, revisions, `ValidatePath`, `AdjustPath` (P4) |
| Exercise Engine | AI-generated and versioned topic sets, nine kinds, deterministic or AI-assisted grading and saved attempts. V1 attaches evidence to attributable attempts. [exercise command](../apps/learning/backend/internal/learningpath/commands/exercise.go), [attempt](../apps/learning/backend/internal/learningpath/commands/attempt.go) | Hidden answer keys/reveal, item-aware generation, rubric and mistake taxonomy (P5) |
| Knowledge Evidence | V1 append-only `learning_evidence` written with an attempt; Quick Screening adds assessment attempts. Source and practice mode are separate. [migration 0036](../apps/learning/backend/migrations/0036_learning_loop.sql), [migration 0037](../apps/learning/backend/migrations/0037_adaptive_assessment.sql), [loop write](../apps/learning/backend/internal/learningpath/commands/loop.go) | Backfill historic attempts and broader sources (P2/P5/P9) |
| Mastery | `learner_item_state` derives `learning`, `practiced`, `mastered`, `relearning` from weighted, time-sensitive evidence; independent recall/production is needed for mastery. Assessment evidence informs a screening estimate but does not directly advance mastery. [pure rules](../apps/learning/backend/internal/learningprofile), [progress model](learning/PROGRESS_MODEL.md) | Full per-skill estimates, rebuild command and calibrated confidence |
| Review Scheduler | V1 item ladder 1/3/7/14/30 days, due activities and answer-without-key reviews. Notebook cards retain their separate 1/2/4/7/15/30/60 Leitner ladder. [loop rules](../apps/learning/backend/internal/learningprofile), [activities](../apps/learning/backend/internal/learningpath/commands/activity.go) | Full review queue/adaptation and mistakes (P6) |
| Overview Dashboard | V1 `/learning/path/dashboard` reports state/review/evidence totals and activities. The working-tree Overview combines it with the active path and lesson totals/history; the new-learner layout shows explicit empty states, a completed program offers extra practice, and due review exercises submit `activityId`. [query](../apps/learning/backend/internal/learningpath/queries/loop.go), [Overview](../apps/learning/frontend/src/features/home/components/overview-screen.tsx), [new-learner state](../apps/learning/frontend/src/features/home/components/new-learner-dashboard.tsx), [review UI](../apps/learning/frontend/src/features/home/components/review-activity.tsx) | Full P8 analytics, real skill estimates, insight rules and final design alignment |
| AI Gateway / Tutor | User key settings, multi-provider completion, JSON parse/validate/retry, usage accounting and existing chats. Streaming chat profile is DeepSeek-only. [AI plumbing](../apps/learning/backend/internal/learningpath/commands/agent.go), [providers](../apps/learning/backend/internal/llm), [AI contracts](learning/AI_CONTRACTS.md) | Versioned shared contracts, stronger gateway failure handling, tutor context/evidence (P9) |

The [Implementation Status](learning/IMPLEMENTATION_STATUS.md) records V1 as verified
in production, P3 Quick Screening as locally verified, P0 hygiene open, and P1/P2/P4-P10
as not started. “Not started” means the **full phase**, even where V1 implemented a
small useful subset. Migrations 0037–0039, Quick Screening code, new Overview/onboarding UI
and their documentation are uncommitted at this snapshot; do not equate the V-004 local
gate with a deploy.

## 5. Current data models

The authoritative schema is the [goose migrations](../apps/learning/backend/migrations)
and [sqlc SQL](../apps/learning/backend/queries); the
[data model guide](learning/DATA_MODEL.md) maps legacy tables and proposed additions.
Core entities: `users`; `learning_ai_settings`; `learning_teachers` (language profile);
`learning_path_tests` → questions → answers; `learning_path_jobs`; `learning_paths`
→ topics → exercises/versions → attempts; `learning_path_lessons`;
`learning_rule_practice`; `learning_knowledge_items` (`kb_items` compatibility view);
`learning_evidence`; `learner_item_state`;
`learning_activities`. Quick Screening extends existing tests/questions/answers and
evidence via migration 0037; migration 0039 adds `learning_assessment_sections` and
`learning_assessment_section_questions` without introducing a parallel test table.
Notebook entities (`learning_topics`, `learning_entries`, captures/sets/reviews) and
chat entities (`learning_chats`, messages, daily usage) stay separate. Profile wizard
answers in localStorage are a frontend model, not a persisted backend profile yet.

## 6. API contracts

Base path `/api/v1`; success envelope `{code:0,message,data}`, camelCase fields,
UUID ids, RFC 3339 UTC timestamps, localized errors. Sessions protect path, test,
assessment, dashboard, AI and chat routes; scoped extension tokens reach only
specified notebook/capture routes. Ownership failures return 404.
[Contract rules and endpoint index](learning/API_CONTRACTS.md),
[canonical payload reference](../apps/learning/backend/docs/learning-api.md),
[route registration](../apps/learning/backend/internal/learningpath/routes.go).

Key path contracts: `GET/POST/PATCH/DELETE /learning/path/teachers`; `POST
/learning/path/tests` (legacy or opt-in `purpose=quick_screening`); `GET
/learning/path/tests/:id`; `POST .../:id/answers`; `POST
.../:id/assessment/:action` and `GET .../:id/results` for Quick Screening;
`POST .../:id/analyze` + `GET /learning/path/jobs/:id` for background builds;
`GET /learning/path?language=`; `POST /learning/path/exercises/:id/attempt`
(optional `activityId` for review); `GET /learning/path/activities?language=`;
`GET /learning/path/dashboard?language=`. Existing response shapes stay compatible;
the frontend's closed zod enums remain a rollout hazard for new values.

## 7. AI integration

The learner configures a provider/model/key; the key is sealed at rest and never
returned by the API. `Agent.completeJSON` opens the key, invokes `llm.Client`, extracts
JSON, validates it, retries malformed output once, then lets a command handler persist
validated data. Calls are metered by purpose. AI creates missing assessment question
**content**, program wording/selection or exercises and grades some free-text answers;
the Quick Screening policy chooses its own targets and placement verdict. No AI call
has database or tool access. See [AI contracts](learning/AI_CONTRACTS.md),
[gateway code](../apps/learning/backend/internal/llm),
[agent wrapper](../apps/learning/backend/internal/learningpath/commands/agent.go).

## 8. Level determination

Legacy placement climbs CEFR bands using AI-written batches and an AI examiner verdict
bounded by backend guardrails; final tests confirm program completion. A learner can
also claim a start level. Opt-in `grammar-quick-v1` starts at A2 by default or a
self-reported band, selects curated questions by band/category/item, grades against
private server keys, and moves up/down/holds by versioned deterministic thresholds.
It needs enough independent responses and a replicated/bracketed boundary, otherwise
returns a provisional range or partial result. It measures grammar only; the program's
legacy `levelReached` uses the lower range boundary or A1 if unknown. No CEFR
certification is implied. [Quick Screening method](learning/QUICK_SCREENING.md),
[policy implementation](../apps/learning/backend/internal/learningpath/assessment/policy.go),
[legacy grading](../apps/learning/backend/internal/learningpath/placement).

## 9. Learning Path generation

An assessment or claimed-level test is stored, then `/tests/:id/analyze` creates an
idempotent job. The worker builds and saves the program, then prefetches first-topic
exercises. For supported library pairs, the legacy route asks AI to choose existing
topic ids and validates them; unsupported pairs use AI-authored whole programs.
The Quick Screening route uses its item outcomes (strong/suspected gaps, checks,
strengths), a small explicit prerequisite map and learner preferences to order
canonical library topics. The path is still static after creation; full graph-based
generation and path revisions are future P4 work. [legacy/branching analyzer](../apps/learning/backend/internal/learningpath/commands/analyze.go),
[screening planner](../apps/learning/backend/internal/learningpath/commands/assessment_plan.go),
[learning-engine design](learning/LEARNING_ENGINE.md).

## 10. Known limitations

- The JSON library covers English/Spanish in five explanation pairs, A1-B2; it has
  no full prerequisite graph, versioned content or C1/C2. Other pairs can receive
  AI-authored rules. [Knowledge Base](learning/KNOWLEDGE_BASE.md).
- Program exercise keys reach the browser before the first answer, so guided
  attempts are assisted and cannot alone establish mastery. [V1](learning/VERTICAL_SLICE_V1.md#9-known-limitations-to-carry).
- Evidence predating V1 was not replayed. Clearing program history can leave
  evidence; teacher deletion cascades it. Full reset semantics need a product
  decision ([D-011](learning/DECISIONS.md#before-phase-2-3)).
- Quick Screening is provisional grammar screening, not calibrated multi-skill
  assessment; other skill estimates remain unknown. [Method](learning/QUICK_SCREENING.md).
- Wizard answers are local to the browser. A new learner with no AI key cannot
  build every path; deterministic features still work.
- P0 test reruns on one DB can hit retained chat token counters (TD-03); frontend
  unit tests and CI e2e remain open. [Testing](learning/TESTING_STRATEGY.md).
- Production data shape for future backfills is unknown (R-01); frontend closed
  zod enums create deployment risk for new values (R-07).

## 11. Architectural decisions

The [decision log](learning/DECISIONS.md) is authoritative. In particular:
D-024 puts a thin exercise → attempt → evidence → state → review → dashboard loop
before horizontal phases; D-025 separates exercise source from free/tracked/assessment
practice modes; D-026 persists typed activities independently of sessions. D-030/031
encode those distinctions in schema; D-032 requires independent evidence for mastery;
D-034 defines deterministic Quick Screening and legacy compatibility. D-027 keeps
lowercase wire values; D-028 defines mastery/review vocabulary; D-029 chooses the
configurable 1/3/7/14/30 item ladder. D-100..D-110 preserve the stack, monolith,
backend authority, learner-owned AI key and additive rollout rules. Several broader
phase decisions remain marked `AWAITING`; their recommended designs are **not**
completed work. The owner has now authorized creation of the root `AGENTS.md`
(D-005); this document does not settle other pending decisions.

## 12. Next work and navigation map

The local working tree now has the eight-section sidebar and mobile menu from
the Overview design. The [canonical local design handoff](learning/frontend/README.md)
contains the runnable prototype, screenshots and visual rules; its
[navigation map](learning/frontend/NAVIGATION_HANDOFF.md) separates intended
interactions from implemented behavior. The hosted MagicPath preview is an
outdated historical reference.
The implementation is in
[`app-shell.tsx`](../apps/learning/frontend/src/components/app-shell.tsx).

| Tab/concept | Present destination and responsibility | Dependency / next step |
| --- | --- | --- |
| Overview | `/learning/overview`: learner home, setup state or continue/review/weekly summary | Aggregates path, activities, lesson history; design alignment now, full P8 later |
| Learning Path / Teacher | `/learning/path`: language teacher, program, topics and exercise runner | Begins after onboarding/assessment/build; P4 later |
| Knowledge Base | `/learning/knowledge`: existing Rule Book with real language/category/level/search filters; notebook `/learning` is labelled as a separate personal collection | Full KB P1 and Rule Book versioning later |
| Practice | `/learning/practice`: due tracked reviews, program activity and separate free Rule Book/notebook practice | Broader P5/P6 activity types later |
| Assessments | `/learning/assessment`: opt-in Quick Screening/manual starting point; legacy final test remains on the path | Broader P3 |
| Progress | `/learning/progress`: real program topic, knowledge item, evidence and review totals for the selected language | Skill estimates and full P8 analytics remain future work |
| AI Tutor / Chat | `/learning/chat`: existing free teacher chat; topic/rule chats are contextual | P9 attaches KB context and validated evidence |
| Settings | `/learning/settings`: AI provider, account and related controls | Keep learner key and language management discoverable |

**Start a new plan** now uses `POST /path/reset` to archive the active program,
then returns to a choice of available assessment, claimed level or A1. Previous
learning history is preserved. The separate scope of deleting history and
progress remains a product decision. No test-user data was changed in this UI
pass. After the UI/navigation pass, the planned engineering
backlog continues with P3-17 (vocabulary objectives and evaluation, subject to P1 lexical
content/identity), while P0-05 (re-runnable DB tests), P0-11 (frontend enum tolerance), P1/P2
and the remaining P3-P10 work stay open in the [implementation plan](learning/IMPLEMENTATION_PLAN.md)
and [live status](learning/IMPLEMENTATION_STATUS.md).
