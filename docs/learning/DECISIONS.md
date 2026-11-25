# Decision log

Format: `ID · title · status`. Statuses: `ACCEPTED` (decided by the owner, in the plan or
earlier work), `DECIDED-REVERSIBLE` (made by the agent during the audit because work had to
start; trivially reversible; the owner may overrule), `AWAITING` (needs the owner before the
named phase), `SUPERSEDED`.

Add new decisions at the end; never renumber; change a status with a dated line.

---

## A. Already decided (do not re-litigate)

| ID | Decision | Source |
| --- | --- | --- |
| D-100 | Keep the stack: Go + Echo + sqlc + goose, Next.js + shadcn/ui, Postgres; no replacement without approval | plan sections 2, 9 |
| D-101 | Modular monolith; no microservices, no broker | plan sections 6, 9 |
| D-102 | CEFR is a reference scale, not the architecture | plan section 4 |
| D-103 | Backend owns state; AI assists and is validated before persistence | plan section 8 |
| D-104 | AI is bring-your-own-key per learner; provider-independent client | existing product |
| D-105 | Library rules/words/goals are written and checked by people; AI picks/orders/exercises | `curriculum.go`, owner 2026-11 |
| D-106 | Package-by-feature, handlers called directly (no bus), sqlc as repository, `apperr` | owner 2026-07 ("form B"), `ARCHITECTURE.md` section 2 |
| D-107 | Rule practice keeps **no history** (a sitting expires 10 min after the learner leaves) | owner 2026-11-12 |
| D-108 | Push to `main` deploys to production; migrations run on deploy; only the changed app is deployed | CI design |
| D-109 | Pin tool versions; never `:latest`/`@latest` | owner standing rule |
| D-110 | Write backend tests in the same change as the feature | owner standing rule |

## B. Made during the audit (reversible)

### D-001 · Documentation lives in `docs/learning/` · DECIDED-REVERSIBLE
The plan asked for `docs/`. The repository is a monorepo (budget + learning), and a root
`docs/PRODUCT_VISION.md` would be ambiguous. `docs/learning/` also sits outside the CI path
filter `apps/learning/**`, so a docs-only push triggers **no production deploy**.
*Reversal:* `git mv docs/learning/* docs/` and fix the README links (one commit).

## C. Awaiting the owner

Each item states the question, options, a recommendation, what it blocks, and the default
the agent will assume **only for design work, never for irreversible steps** if no answer
comes. Items are ordered by when they block.

### Before Phase 1 starts

**D-002 · Who writes the code for each phase · ACCEPTED (2026-11-25), updated responsibilities.**
The owner delegates the implementation of the agreed backend phases to the agent. The earlier
standing preference "the learning backend is written by the owner" no longer blocks agreed work.
The agent may, on its own: analyse existing code, implement agreed functions, refactor locally,
write backward-compatible migrations, add tests, update documentation, fix bugs it finds,
implement API contracts. A **separate approval** is required before: destructive migrations,
deleting user data, removing existing functionality, replacing the technology stack, a
substantial architecture change, changing previously agreed product requirements. Work is done
in stages. *Replaces:* the owner-writes-the-backend preference (for agreed phases only) and this
log's earlier default "no implementation without an explicit go per phase".

**D-003 · Commit and push policy for phase work · DECIDED-REVERSIBLE (2026-11-25, interpretation of D-002).**
Pushing `main` deploys. D-002's delegation plus the standing rule "commit and push without
asking" are read as covering additive work. Kept conservative: one phase step per push, only
after the full local gate (migrate up, vet, tests on fresh databases, eslint/tsc if the
frontend changed); CI and the deploy are watched afterwards and the result is reported;
migrations are additive with a working Down; a destructive step still needs D-002's separate
approval. The owner can tighten this at any time.

**D-020 · Production data access for migration rehearsal (R-01) · AWAITING.**
The audit could not see production row counts or content shapes. *Options:* (a) owner runs a
provided read-only SQL report and pastes counts; (b) owner provides an anonymised dump to
restore locally; (c) rehearse only on synthetic data. *Recommendation:* (a) then (b) for the
Phase 1/2 backfills. *Blocks:* the first backfill (Phase 1 task P1-09, Phase 2 P2-05).

**D-007 · Learner-profile root = `learning_teachers` · AWAITING (recommend accept).**
Keep the existing table as the per-language profile and add columns/related tables rather
than creating a parallel `learner_profiles` table. *Pro:* existing teacher profiles and
history preserved by construction; no data copy; one concept ("teacher" in the UI =
profile). *Con:* the name is product-specific; renaming later is a cheap DB rename. *Blocks:* Phase 2.

**D-008 · Knowledge-base authoring source · AWAITING (recommend accept).**
JSON in git stays authoritative; the DB is the runtime store, filled by an idempotent import;
no content-admin UI in this plan. *Alternative:* DB-authoritative with an admin editor (more
power, much more work, and conflicts with "reviewed by people in PRs"). *Blocks:* Phase 1.

**D-009 · Enum policy for new schema · AWAITING (recommend accept).**
New columns use `text` + `CHECK`, not Postgres enums (values cannot be removed from an enum;
adding one is irreversible and breaks the frontend's closed `z.enum`s — R-07). New *values* of
existing enums are avoided by using separate flags/columns (`DATA_MODEL.md` B.3).
*Blocks:* Phase 1 migrations.

**D-010 · How the notebook and the teacher/program universes relate · AWAITING.**
Today they share nothing but language codes (`CURRENT_STATE.md` section 4). *Options:*
(a) keep separate for this plan (recommended): notebook SRS unchanged; KB `lexeme`/`phrase`
items are *content* and never write notebook rows; a "save to notebook" action stays UI-level;
(b) mirror notebook cards as KB `phrase` items so captured phrases join the adaptive loop
(valuable, large: needs matching, dedup, ownership rules); (c) merge. *Decides also:* whether
vocabulary words become `lexeme` items in Phase 1 or later (K-1). *Blocks:* Phase 1 scope
(lexemes yes/no); Phase 6.

**D-018 · Extension/capture API freeze · AWAITING (recommend accept).**
Treat `/learning/captures`, `/entries`, `/topics` as frozen contracts (tokens live in users'
browsers). *Blocks:* nothing now; protects Phases 1-6.

**D-021 · Language scope of the implementation · AWAITING.**
Phase 1-9 work on the data that exists: English and Spanish, explained in Russian/English/
Spanish (5 pairs), A1-B2. *Question:* is C1/C2 content, a third studied language, or a new
explanation language required before Phase 10? *Recommendation:* no; design generically,
ship with what exists. *Blocks:* Phase 10 planning only.

### Before Phase 2-3

**D-011 · What "delete history" may erase · AWAITING.**
`DELETE /path/history` hard-deletes closed programs, topics, exercises, attempts, lessons.
Once evidence exists the plan's rule is "never silently discard learning history".
*Options:* (a) keep as is but make the evidence rows survive (they reference no deleted row)
and say so in the UI; (b) deletion becomes "hide" (soft delete) for the learner while
evidence/estimates stay; (c) deletion also erases evidence (user's right to erase; estimates
rebuilt). *Recommendation:* (a) with an explicit confirmation naming what is erased; evidence
erased only by teacher/account deletion (cascade). *Blocks:* Phase 2 (evidence table FKs),
Phase 5.

**D-022 · What "level" means for building a program · AWAITING.**
Programs are built from one `levelReached` today. With per-skill estimates (`ASSESSMENT_ENGINE.md`
section 3.6) the legacy `levelReached` must be defined: **weakest core skill** (recommended;
conservative, never skips foundations), median, or per-skill levels fed to path generation
(Phase 4 does that regardless). *Blocks:* Phase 3 compatibility field.

**D-017 · Shared bank of AI-written assessment items · AWAITING.**
Reusing validated AI-written items across learners cuts cost and enables item analysis, but
items written on user A's key serve user B and need a no-personal-data guarantee.
*Options:* (a) per-test only (today); (b) shared bank restricted to items generated from
KB items and verified by item analysis (recommended later); (c) shared immediately.
*Default:* (a). *Blocks:* Phase 3 item-quality plan only.

**D-023 · Level of detail of the tutor on non-DeepSeek providers · AWAITING.**
Streaming chat exists only for DeepSeek (`chatProfiles`). Tutor evidence (Phase 9) inherits
that. *Options:* add chat profiles for Anthropic/OpenAI/Kimi (work in `llm`), or document
tutor evidence as DeepSeek-only. *Blocks:* Phase 9.

### Before Phase 4-6

**D-013 · Content ownership: prerequisites, C1/C2, new pairs · AWAITING.**
Prerequisite graphs (English 85 nodes, Spanish 91) do not exist and are the main content
effort (`KNOWLEDGE_BASE.md` section 5). *Options:* (a) owner/linguist authors them; (b)
offline model-drafted edges, human-reviewed in PRs, never at runtime (recommended); (c)
skip prerequisites (Phase 4 then validates nothing — not recommended). *Blocks:* Phase 4.

**D-015 · Reference sources and licences · AWAITING.**
Which sources may the KB cite and import from (CEFR descriptors, commercial grammar/vocabulary
profiles, curricula)? Citation is always allowed; copying lists/text needs a licence check
per source. *Blocks:* Phase 1 `kb_sources` content (can ship with `authored` only).

**D-016 · Review scheduler model · AWAITING (recommend accept).**
Start with the explainable stage ladder (generalising today's Leitner 1,2,4,7,15,30,60);
revisit a data-driven stability model (FSRS-style) once >= 10,000 review events exist.
*Blocks:* Phase 6.

**D-012 · Roles and the "teacher viewer" · AWAITING (separate stream).**
A separate plan (admin area, `account_roles`, read-only teacher link) exists and is not
started. This plan assumes **no roles** and keeps handlers role-agnostic so the viewer can be
added at the controller layer. *Question:* start the roles stream before, during or after
Phase 8 (the dashboard is the natural thing to share)? *Blocks:* Phase 8 sharing only.

**D-014 · Rollout flags · AWAITING (recommend accept).**
Per-feature flags as config values (env), default off, flipped per deploy — e.g.
`LEARNING_KB_READ=db`, `LEARNING_ASSESS_ENGINE=backend|shadow|ai`, `LEARNING_PATH_ENGINE=…`.
No flag service. *Blocks:* Phase 1 read-switch.

### From the backend brief (2026-11-25, see `BACKEND_BRIEF_FIT.md`)

**D-024 · Order of delivery · ACCEPTED (2026-11-25).**
A thin end-to-end learning loop comes before the horizontal phases are finished. First vertical
slice (V1): (1) a grammar rule from the existing library, (2) get or generate its exercise, (3)
the learner's answer, (4) the check, (5) the attempt saved, (6) evidence formed, (7) knowledge
state updated, (8) a review scheduled, (9) the minimal dashboard updated. All phases stay; only
the order of the tasks needed for V1 changes. No complex algorithms and no advanced analytics
before the loop is proven. Concrete plan: `VERTICAL_SLICE_V1.md`. *Replaces:* the layered order of
`IMPLEMENTATION_PLAN.md` section 5 (kept as the order of the *rest* of the work).

**D-025 · Rule Book practice and learning evidence · PARTIALLY ACCEPTED, my proposal revised (2026-11-25).**
Rule-book practice is **not** excluded from evidence by nature. The *source* of an exercise and
the *mode* of practice are separate. Modes: `FREE_PRACTICE` (keeps D-107 exactly: no permanent
answer history, no mastery change, no official evidence, no effect on the path or the review
schedule), `TRACKED_PRACTICE` (stores attempts, forms evidence, may update mastery, may write
mistakes, may affect the schedule, accounts for hints and retries), `ASSESSMENT` (formal check:
stores attempts and evidence by documented criteria, separates independent from assisted
answers, may update knowledge estimates and recommendations). Sources: `RULEBOOK`,
`LEARNING_PATH`, `STUDY_SESSION`, `REVIEW`, `AI_TUTOR`, `ASSESSMENT`. Existing free practice is
unchanged. Tracked practice is added later as a separate mode; V1 implements neither rule-book
tracking nor assessment mode. *Replaces:* my option (a) "rule practice stays outside evidence";
*extends* D-107 (it now defines `FREE_PRACTICE`).

**D-026 · Typed learning activities · PARTIALLY ACCEPTED, my proposal revised (2026-11-25).**
The topic-based Learning Path stays. Typed activities do **not** live only inside a Study Session.
Seven concepts are kept apart: Knowledge Item, Learning Path Topic, Learning Activity, Exercise,
Study Session, Attempt, Evidence. A Topic may contain or reference typed Activities (`LEARN`,
`PRACTICE`, `REVIEW`, `APPLY`, `ASSESS`). A Study Session selects existing activities of the path
and may add review activities. Exercises need not be generated ahead for future activities. An
Activity may store a learning objective, type, linked knowledge, status and generation
parameters; its exercises can be created when it starts. **Completion is stored independently of
any session.** No duplicate entities where the architecture already supports the need. V1: a
minimal `PRACTICE` and `REVIEW`. *Replaces:* my recommendation (a) "types only in the session plan".

**D-027 · Wire naming of enums · DECIDED-REVERSIBLE.** The brief writes UPPER_CASE; the existing
API and database use lowercase snake (`active`, `choice`). Follows "use existing API conventions":
lowercase on the wire, UPPER_CASE only in prose.

**D-028 · Mastery vocabulary and review state · DECIDED-REVERSIBLE (recommend accept).** API names
follow the brief: no row = `unknown`; row without evidence = `not_started`; `learning`;
`practiced`; `mastered`. The internal `relearning` is shown as `learning` with `lapsed: true`.
The review state (`not_scheduled/scheduled/due/overdue`) is derived at read from `due_at`, never
stored. All as `text` + `CHECK` (D-009).

**D-029 · Review ladder defaults · APPLIED IN V1 (awaiting the owner's confirmation).** The brief suggests 1/3/7/14/30 days; the notebook
uses 1/2/4/7/15/30/60. *Recommendation:* the ladder is a configuration table; KB items default to
the brief's values, the notebook is untouched; re-tune only on measured data. Refines D-016.

(D-002 is answered above: accepted, 2026-11-25.)

### Decisions made while preparing V1 (reversible; implement D-024..D-026)

**D-030 · Evidence vocabulary · DECIDED-REVERSIBLE.** Implements D-025. A `learning_evidence` row
carries `source` (`rulebook|learning_path|study_session|review|ai_tutor|assessment`),
`practice_mode` (`tracked|assessment`; `free` never writes a row), a reference to what produced
it (`ref_type`, `ref_id`; unique, so one attempt gives at most one row), `mode`
(`recognition|recall|production`), `outcome`, `weight`, and `assisted`. All `text` + `CHECK`
(D-009). V1 writes `learning_path` and `review` rows with `tracked`.

**D-031 · Learning Activity model · DECIDED-REVERSIBLE.** Implements D-026. One table
`learning_activities` (type, status `open|completed|skipped`, reason, goal, item, topic, params),
closed type list, partial unique indexes (one open review per item, one open practice per topic).
Rows are materialised lazily by the read that lists them; a finished practice activity is closed
there from the topic's own `completed_at`; a review activity is completed in the attempt's
transaction. No session is involved.

**D-032 · Mastery needs independent evidence · DECIDED-REVERSIBLE.** Program exercises show their
key before the first answer (G-07), so their evidence is `assisted`. It can take an item to
`practiced` but **only an independent answer** (a review, whose exercise never carries its key,
later a hint-free tutor or assessment answer) can make it `mastered`.

**D-033 · V1 scope and deferrals · DECIDED-REVERSIBLE.** The list of what V1 leaves out is in
`VERTICAL_SLICE_V1.md` section 8. Notably no frontend change (the design arrives separately), no
replay of past attempts, no new response shapes for existing endpoints.

### Replaced or extended by the 2026-11-25 answers

| Earlier decision | Now |
| --- | --- |
| Owner writes the learning backend (standing preference) | **Replaced** for agreed phases by D-002 |
| D-002 default "no implementation without a per-phase go" | **Replaced** by D-002 (staged, separate approval only for destructive / stack / scope changes) |
| Layered order of `IMPLEMENTATION_PLAN.md` section 5 | **Extended** by D-024: V1 first, then the rest in the old order |
| D-107 "rule practice keeps no history" | **Kept and renamed** `FREE_PRACTICE`; **extended** by D-025 with tracked and assessment modes |
| My D-025 proposal (outside evidence) | **Replaced** by the three modes |
| My D-026 proposal (types in the session plan only) | **Replaced** by persisted activities |
| D-016 (review ladder shape) | **Refined** by D-029 (default 1, 3, 7, 14, 30 days) |
| D-007, D-008, D-009, D-014 | unchanged; V1 follows them (teacher is the profile root, JSON stays authoritative, `text`+`CHECK`) |

### Housekeeping

**D-004 · The stale `apps/learning/backend/CODING_STYLE.md` · AWAITING.**
Describes hexagonal/gin/wire/MySQL, none of which exist here. *Options:* delete; replace
with a 10-line pointer to `ARCHITECTURE.md` section 2 (recommended); leave. *Blocks:* nothing.

**D-005 · Create an agent instruction file · AWAITING.**
No `AGENTS.md`/`CLAUDE.md` is tracked. *Recommendation:* add a short root `AGENTS.md` that
points to `docs/learning/README.md` and `ARCHITECTURE.md` section 2 and states the three hard
rules (push = deploy; additive migrations; no secrets in the repo). *Blocks:* nothing.

**D-006 · New package names and extraction timing · AWAITING (recommend accept).**
`learningkb`, `learningprofile`, `learningassess`, `learningexercise`, `learningreview`,
`learningsession`, `learningprogress`, `learningtutor`, `learningaicontract`; extract per
phase, do not move existing code up front (`ARCHITECTURE.md` section 1.1).

**D-019 · Frontend test tooling · AWAITING.**
CI runs eslint + tsc only. *Options:* add a unit runner (vitest) for schema contract tests
(needs a new dev dependency — approval); put Playwright e2e into CI (needs a seeded
backend job); neither (type-level fixtures only). *Recommendation:* vitest for zod contract
tests first, e2e later. *Blocks:* P0-06.

## D. Log

| Date | Entry |
| --- | --- |
| 2026-11-25 | Audit executed; D-001 made; section C compiled; no code changed |
| 2026-11-25 | Backend brief compared with code and plan (`BACKEND_BRIEF_FIT.md`); D-024..D-029 added, D-002 restated; no code changed |
| 2026-11-25 | Owner accepted D-024, D-002, revised D-025 and D-026; D-003 interpreted; D-030..D-033 added; `VERTICAL_SLICE_V1.md` written; implementation of V1 started |
