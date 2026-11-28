# Fit of the "Backend Implementation" brief to this repository

Date: 2026-11-25 · against commit `8ad4470` (docs) / `4662ad7` (code) · analysis only, nothing implemented.

The owner supplied a second brief, "Backend Implementation — Adaptive Language Learning
Platform" (20 sections). It is written for an implementing agent ("act as a senior backend
engineer"), unlike the first master plan, which is organised by phase. This file answers one
question: **how close is that concept to what the code does and to what `IMPLEMENTATION_PLAN.md`
already says, and what has to change.** Section numbers below (§N) are the brief's.

## 1. Verdict

The concept is the same product as the master plan: same domains, same hard rules (modular
monolith, backend owns state, AI output is untrusted, no mastery from one multiple-choice answer,
unknown stays unknown, no fabricated textbook references, no destructive migration without
approval). **No part of it contradicts the existing architecture** and none requires replacing the
stack or the data. Of 19 rows in the table below: **6 are already covered** by code or plan,
**11 need an addition or a rename** in the plan (all small, listed in section 4), and **2 are real
conflicts** (section 3) that need the owner: the order of work (§18) and who writes the code (§20).

Two things in the brief are *further along in the code than the plan assumed*: the exercise kinds
(nine kinds exist, including `listen` and `speak`, which the brief treats as future extension
points) and the explanation setting (`learning_teachers.explain_in_target`, migration 0027, already
exists as a boolean). One thing is *weaker in the code than the brief assumes*: a failed AI call
aborts the whole attempt and stores nothing (section 2, §6).

## 2. Fit, section by section

Legend: **OK** = in the code or already in the plan as written · **ADD** = plan needs an addition,
a rename or a new task (section 4) · **DECIDE** = owner decision (section 3).

| § | Brief asks for | In the code today | In our plan | Verdict |
| --- | --- | --- | --- | --- |
| 1 | Incremental, modular monolith, preserve behaviour, update docs per phase | Package-by-feature monolith; additive migrations | `IMPLEMENTATION_PLAN.md` section 3 rules | **OK** |
| 2 Knowledge Base | Items with type/category/CEFR/version/publication; typed relations; localised explanations apart from the canonical item; sources without fabricated unit/page; migrate the Rule Book | Embedded JSON, topic-level, ids unique per file, no relations/versions/sources. Rule fields cover formation/examples/mistakes; no separate meaning/usage/comparisons/exceptions. No prompt cites a unit or page; the prompts name a book only as a *style* cue (`practice_prompt.go`: "in the manner of Raymond Murphy's English Grammar in Use") | P1: extend `learning_knowledge_items`, add `learning_knowledge_item_versions` per explanation language, relations, sources, importer, parity test | **ADD**: relation `extends` (ours: prerequisite/related/contrasts_with/part_of); per-kind `body` schema with the brief's eleven sections (a content task more than a code task); text **search** endpoint; rule "AI never emits unit/page, validators reject them" |
| 3 Learner profile | Per user+language profile; explanation mode NATIVE/TARGET/MIXED; goals, weekly target, preferred duration; skill profiles with unknown; per-item progress | `learning_teachers` = per-language profile with `translation_language` and `explain_in_target` (boolean); lesson length fixed at 30 min; no skill estimates, no item progress | P2: teacher is the root (D-007); `learner_skill_estimates` (history + `basis`), `learner_item_state`, evidence ledger | **ADD**: `explanation_mode` as text+CHECK with three values (the boolean maps to NATIVE/TARGET; MIXED is new); goals, weekly target and preferred minutes as profile columns (where weekly target lives today was not verified) |
| 4 Assessment | Five modes; backend-owned stop rules; partial, resumable, manual start, from zero; per-question item and skill; assisted vs independent; no AI-assigned final level | AI examiner decides climb/stop under guard rails; choose-level exists (`source=chosen`); skills on questions exist | P3: Bayesian posterior per skill, stop rules, shadow mode, purposes `placement/final/skill/diagnostic/reassessment`, `stop_reason` | **OK**; add a light `quick` purpose to map QUICK_SCREENING. The brief's "no arbitrary AI level" is exactly gap G-05 |
| 5 Learning path | Deterministic planner over canonical items; activity types LEARN/PRACTICE/REVIEW/ASSESS/APPLY; reason per activity; auditable revisions; configurable priority | Path = ordered *topics* picked by AI from the library; no reasons, no revisions | P4: `reason`, `priority`, revisions, `AdjustPath`, `ValidatePath`, weights gap .40 / goal .25 / unlock .15 / recur .15 / cost .05 | **ADD**: two priority factors missing (assessment uncertainty, skill practice imbalance; review urgency is handled by the review queue). **DECIDE**: typed activities vs topics (D-026) |
| 6 Exercises | Nine types, extension points, type-specific schemas, accepted variants, rubric, hints, versions, five outcomes, no incorrect-on-AI-failure, first attempt vs retry, reveal not mastery | Nine kinds exist (below). Flat row: `kind`, `text`, `prompt`, `answer`, options. Attempt = `correct` boolean + `mistakes` int + `feedback`. **AI failure aborts the attempt: nothing is stored**, so a deterministic wrong answer is lost when only the explanation call failed. No hints/reveal/retry number/idempotency | P5: schema v2, item links, `attempt_no`, `hints_used`, `mode`, reveal endpoint, rubric C-2, evidence in the same transaction | **ADD**: `outcome` (`correct/incorrect/partial/needs_review/unavailable`) as a new text column next to `correct` (the frontend reads `correct`; do not touch the enum, R-07); store the attempt with `unavailable` instead of aborting; submission idempotency key |
| 7 Mistake journal | Categories incl. collocation and comprehension; normalised patterns; stylistic preferences are not errors; targeted review | Prompt already excludes typing slips (capital letter, spaces, full stop) from mistakes; no journal | `learner_mistakes` (P5/P6), closed taxonomy (form, agreement, tense_aspect, word_order, article_determiner, preposition, lexical_choice, spelling, pronunciation, register, omission, …) | **ADD**: category + subtype (the brief's eight are the category, ours are subtypes; add `collocation`, `comprehension`); a `pattern_key` column for grouping |
| 8 Mastery | States UNKNOWN/NOT_STARTED/LEARNING/PRACTICED/MASTERED; review state separate; six evidence kinds; configurable minimums; degradation; no calibrated-probability claims | Nothing for KB items | `unseen/introduced/learning/practiced/mastered/relearning`; evidence modes recognition/recall/production + hints; transition table; lapse -> `relearning` | **ADD**: API vocabulary (D-028). Review state NOT_SCHEDULED/SCHEDULED/DUE/OVERDUE is **derived at read** from `due_at`, never stored (cannot go stale) |
| 9 Review scheduler | Configurable ladder, default 1/3/7/14/30; pluggable; not a flashcard algorithm blindly | Notebook Leitner 1/2/4/7/15/30/60 (cards only) | P6, D-016: explainable ladder, FSRS later | **OK**; default values decided in D-029 |
| 10 Sessions | States CREATED/IN_PROGRESS/PAUSED/COMPLETED/ABANDONED; 15/30/45/60 min; pause history; active vs wall time; resume; no duplicate completion | `learning_path_lessons`: status `active/finished`, `paused_at`, `paused_seconds`, `pause_count`; one active per user (partial unique index); `Finish` on a finished lesson changes nothing (verified); fixed 30 min | P7: `planned_minutes`, `active_seconds`, `plan`, heartbeats, `study_session_items` | **ADD**: a `state` text column for CREATED/ABANDONED (do not add values to the lesson enum, R-07); a pauses table (today only a count and a total, not a history) |
| 11 Analytics | Four metric families; null when evidence is thin; historical snapshots or reproducible calculation | Scattered scales | P8; append-only evidence makes any past date reproducible (`as_of` parameter) | **OK**; snapshots only if measured to be needed (P8-10) |
| 12 Dashboard | 17 fields, 8 states, deterministic facts, cached AI insight, AI-availability flag | Program/weekly scales only | P8 | **ADD**: persisted/cached insight table; **sequencing**: the response shape must wait for the frontend design the owner will send, so do not freeze it before then |
| 13 AI gateway | Structured outputs, validation, **prompt versioning**, timeouts, bounded retries, usage and **cost**, minimisation, injection defence | Validation + one retry, token usage per hour/purpose, call timeouts, provider retries disabled in the SDK, injection wording in prompts, no names or emails in prompts (grep found none). **No prompt version, no per-generation record, no cost** | Contracts with versions (`AI_CONTRACTS.md`), audit table proposed for the tutor only | **ADD**: cross-cutting task: contract name + version on every call, one `ai_generation_records` table (contract, version, model, tokens, accepted/rejected, reason), cost only where the provider reports it |
| 14 API | Capabilities per area, existing conventions, pagination | Existing endpoints indexed in `API_CONTRACTS.md` | Per-phase planned endpoints | **ADD**: knowledge search, references, goals update, attempt history, `start review`; wire values stay lowercase snake (D-027), not the brief's UPPER_CASE |
| 15 Database | Reuse existing tables, FKs, constraints, idempotency, backfill | Ownership via `user_id` in every query | `DATA_MODEL.md` B, composite FKs `(teacher_id,user_id)`, expand-contract | **OK** |
| 16 Security | Ownership, validation, no duplicate submissions, invalid transitions, rate limits | Ownership by SQL scoping; limiter exists (`ErrThrottled`); **attempts are not idempotent** (a double tap stores two) | Authorization matrix per endpoint | **ADD**: idempotency key on attempts, review results, session completion |
| 17 Tests | Per-domain lists, regression | 262 tests | `TESTING_STRATEGY.md` layers | **OK** |
| 18 Vertical slice first | One end-to-end loop before anything advanced | — | Layered by domain: loop appears only around P6 | **DECIDE** (D-024) |
| 20 Approval / who codes | Agent implements; stops before destructive steps | Owner's rule: backend written by the owner | D-002 awaiting | **DECIDE** (D-002) |

The nine existing exercise kinds against the brief's nine: `choice` = MULTIPLE_CHOICE · `fill` and
`cloze` = FILL_IN_THE_BLANK · `translate` = TRANSLATION · `order` = SENTENCE_BUILDING · `read` =
READING_COMPREHENSION · `write` = WRITING · `listen` = LISTENING · `speak` = SPEAKING (graded by
spoken-phrase similarity in the browser). Not in the program: SENTENCE_TRANSFORMATION and
ERROR_CORRECTION exist only in the **rule practice** (`form/choose/correct/make`), which keeps no
history by the owner's decision (D-107). CONTEXTUAL_PRACTICE, PRONUNCIATION and ROLE_PLAY have no
kind (role play exists as tutor scenarios).

## 3. Where the brief and the repository really disagree

1. **Order of work (§18).** The brief wants the whole loop running first, thin: language ->
   profile -> canonical items -> recommended activity -> exercise -> attempt -> evidence ->
   mistake -> scheduled review -> updated recommendation -> dashboard. Our plan builds by layer
   (KB, profile, assessment, path, exercises, review) and the loop closes only after P6. The brief
   is right: layered delivery postpones the first end-to-end learning on production data, and it
   postpones the first real test of the evidence model, which is the part most likely to be wrong.
   Proposed re-cut in section 5. *(D-024)*
2. **Rule practice vs evidence (D-107).** The owner decided rule practice leaves no history. The
   brief's loop needs persisted attempts. Program exercises already persist, so the slice does not
   need rule practice; it simply stays outside the evidence ledger. Say it explicitly. *(D-025)*
3. **Who writes the code (§20 vs D-002).** The brief addresses an implementing agent, and the
   owner is preparing a frontend design separately. This contradicts the earlier standing rule
   that the learning backend is written by the owner with examples in chat. I do not start
   implementation without an explicit answer. *(D-002, restated)*
4. **Typed activities (§5).** The brief's path is a list of typed activities. Ours is a list of
   topics, and the rest (practice, review, application) is composed per session. Both can satisfy
   "store the reason each activity was selected". *(D-026)*
5. **Documentation paths.** The brief tells the agent to read `docs/IMPLEMENTATION_PLAN.md` and
   six more files. They live in `docs/learning/` (D-001). Either say so in the prompt or move them
   (one `git mv`); an agent that follows the brief literally finds nothing.

## 4. Additions to the plan (small, none changes the architecture)

| # | Addition | Where it lands |
| --- | --- | --- |
| A-1 | relation `extends`; `quick` assessment purpose | P1 (`learning_knowledge_item_relations` CHECK), P3 |
| A-2 | per-kind `body` JSON Schema with the eleven content sections; KB text search endpoint | P1-01, P1-10 |
| A-3 | `explanation_mode` (3 values) on the teacher; goals, weekly target, preferred minutes | P2-01 |
| A-4 | two priority factors: assessment uncertainty, skill practice imbalance | P4-03 |
| A-5 | `outcome` column and "store the attempt even when the explanation call fails" | P5-03 |
| A-6 | idempotency key on attempts, review results and session completion | P5-03, P6-04, P7-04 |
| A-7 | mistake `category` + `subtype`, `pattern_key`; add `collocation`, `comprehension` | P5-06 |
| A-8 | session `state` text column, pauses table, durations 15/30/45/60 | P7-01 |
| A-9 | cached insight table | P8 |
| A-10 | **AI gateway hardening (new cross-cutting task X-1):** contract name+version on every call, `ai_generation_records`, cost where reported; do it before the first new contract ships | before P3/P5 |
| A-11 | validator rule: AI output never carries unit/page/edition references; the `kb_sources` table is filled by people only | `AI_CONTRACTS.md` |

## 5. Proposed re-cut: a thin loop first (needs the owner's yes on D-024)

Keep every phase as written. Change only the order of the first steps so the loop closes early:

| Step | Contents (existing task ids) | What the learner sees |
| --- | --- | --- |
| S0 | P0-05, P0-11, X-1 (hygiene and safety net) | nothing |
| S1 | P1 limited to **grammar items from the existing library** (P1-02..P1-07, P1-09); no lexemes, no prerequisites yet | nothing (shadow compare) |
| S2 | P2-01..P2-04: evidence dual-write from program attempts, item state (coarse, per topic item) | nothing |
| S3 | P6-01..P6-04 reduced: next review per item, review queue endpoint, `outcome`, idempotency (A-5, A-6) | "due reviews" list |
| S4 | P8 reduced: dashboard with the facts that now exist; shape agreed with the frontend design | the new home screen |
| then | P2 rest -> P3 -> P4 -> P5 rest -> P6 rest -> P7 -> P8 rest -> P9 -> P10 as in the plan | |

This is the brief's twelve-step loop with the *simplest* version of each step: topic-level items,
coarse attribution, no assessment change, no path change. Everything built in S1-S3 is a subset of
what the full phases need, so nothing is thrown away.

## 6. New and restated decisions

Recorded in `DECISIONS.md`: D-024 (order: slice first), D-025 (rule practice outside evidence),
D-026 (typed activities), D-027 (wire naming, reversible), D-028 (mastery vocabulary and derived
review state, reversible), D-029 (review ladder defaults), D-002 restated.

## 7. What was and was not checked

Checked in code: exercise kinds and their grading flow, the attempt table, the lesson table and its
idempotency, the teacher table (including migration 0027), the prompts (no unit/page references, no
personal fields found by grep, book names used as style cues only), usage metering, absence of
prompt versions. Not checked: where the weekly target is stored today, the exact frontend contract
for exercises (relevant to A-5), production data (R-01), and whether the frontend design the owner
will send changes any dashboard field.
