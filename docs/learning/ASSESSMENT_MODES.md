# Assessment modes and shared engine (target contract)

**Status: architecture approved by the owner's 2026-11-26 scope update; implementation is incremental.**
The [Quick Screening specification](QUICK_SCREENING.md) describes the currently implemented
grammar-only slice. The [status ledger](IMPLEMENTATION_STATUS.md) is authoritative for
verification. This contract supersedes the Quick-only product scope and the proposed
single-posterior design in [ASSESSMENT_ENGINE.md](ASSESSMENT_ENGINE.md) where they conflict.
It does not claim that the additional modes or skills are available today.

## 1. Mode configuration

One engine owns lifecycle, authorization, question selection, generation, validation,
adaptation, answer evaluation, evidence, gap analysis and aggregation. A mode snapshot
supplies ordered sections, coverage, budgets and stop conditions; a skill blueprint
supplies formats, item sources, evaluator and rubric. Version and snapshot both are
persisted with an attempt, so a resumed assessment uses its original rules.

The shared code slice has a versioned runtime blueprint registry, pure
[`AggregateMode`](../../apps/learning/backend/internal/learningpath/assessment/orchestration.go),
[`TransitionSection`](../../apps/learning/backend/internal/learningpath/assessment/lifecycle.go)
and [question source, writer and evaluator interfaces](../../apps/learning/backend/internal/learningpath/assessment/contracts.go).
Quick uses these contracts for its one grammar section: the bank selects canonical
targets, the AI writer fills exhausted wording, and the keyed evaluator grades answers.
P3-15 adds persisted section rows and a full blueprint snapshot for newly created
grammar Quick attempts. Existing attempts keep their recorded policy; where a full
historical blueprint or section result was never saved, it remains unknown. Other
skill-specific evaluators and multi-section orchestration are still planned.
P3-16 routes grammar target allocation, writer validation, keyed evaluation, adaptive
decisions and result aggregation through [`SectionEngine`](../../apps/learning/backend/internal/learningpath/assessment/engine.go).
The frontend reads persisted grammar-section status and answer count with a legacy-field
fallback and uses the existing Quick lifecycle endpoints. The P3-15 bridge still mirrors
the single section and its legacy parent; this is not multi-section orchestration.

| Mode | Sections | Completion | Current availability |
| --- | --- | --- | --- |
| `quick_screening` | Grammar, vocabulary, reading | Each configured section reaches its minimum evidence or an explicit partial stop; bounded total workload | English/Spanish **grammar-only Quick Screening** is locally verified; full quick scope is pending |
| `full_assessment` | Grammar, vocabulary, reading, listening, writing, speaking; pronunciation only after validated audio scoring | Independent section results; overall completion only after every *required and supported* section completes; user can pause/resume between sections or mid-section | Not implemented |
| `skill_assessment` | One learner-selected supported skill | That skill's evidence and stop rule; each run gets a distinct attempt and appears in history | Not implemented |

Unsupported skills have `unassessed` status and no estimate. A partial section is not a
completed section; the aggregate carries both section status and overall coverage.
Full Assessment must never silently omit a required skill while claiming a complete
full result. Pronunciation is an optional, explicitly labelled extension and cannot be
enabled from a transcript-only pipeline. Manual starting point and start from scratch
remain distinct onboarding actions, outside assessment modes.

### Lifecycle

`created -> in_progress <-> paused -> evaluating -> completed|partial|failed|cancelled`.
The current Quick implementation has durable pause/resume and per-answer writes in
[`commands/assessment.go`](../../apps/learning/backend/internal/learningpath/commands/assessment.go).
The new storage can hold separate lifecycles and result snapshots; the current Quick
grammar section mirrors its legacy parent test during the compatibility rollout.
The independent multi-section workflow remains a target.
Pause never consumes a time budget. A disconnect does not discard accepted answers.
Completing a section writes its result and evidence before the next section is offered;
the learner may leave and resume later. Starting the same open mode returns that attempt;
repeated *finished* skill assessments create new attempts. The backend serializes each
state transition with a row lock, keeps answer submission idempotent, and rejects changed
replays. Historical attempts remain readable.

## 2. Skill blueprints and item contracts

Every question has server-assigned `assessmentId`, `sectionId`, `skill`, `band`, format,
objective/knowledge reference where one exists, opaque version, origin, context and
validation status. The key or scoring rubric stays server-side. The generator receives
backend-selected slots and may only fill content; it may not select a skill, change a
band, issue its own identifiers, decide proficiency or end a test. Generated items are
per-attempt until D-017 is resolved.

| Skill | Required probe families | Evaluation | Minimum source/coverage requirement before enabling |
| --- | --- | --- | --- |
| Grammar | Recognition, independent recall, error correction, transformation, contextual use | Exact/curated alternatives and bounded partial credit | Rule-linked targets across form categories and bands; existing grammar bank can seed the section |
| Vocabulary | Meaning, contextual use, collocations, independent recall | Verified choices and curated accepted forms; contextual rubric for open responses | Lexical objectives and validated keys; grammar objectives tagged `vocabulary_in_context` do **not** count as vocabulary evidence |
| Reading | Main idea, detail, inference, comprehension across passages | Passage-linked verified answers; rubric for open inference | Passage provenance, question-to-passage link, multiple comprehension dimensions |
| Listening | Main idea, detail and inference from playable audio | Answer about the audio content, not text shown beforehand | Stable audio asset, playback, transcript/key hidden during assessment, accessible fallback policy |
| Writing | Independent response to communicative prompt | Versioned criteria and bounded AI rubric score with calibration | Authored prompts/anchors, language/length checks, rubric agreement measurement |
| Speaking | Recorded communicative task; fluency and interaction where supported | Server-side audio-aware evaluation with bounded rubric | Capture, upload, retention/consent policy, speech recognition and audio-aware scoring validated for supported languages |
| Pronunciation | Repeat/read/sound contrast with recorded audio | Validated audio-based articulation/phonetic measure | Audio capture plus appropriate acoustic evaluation; transcript text alone is invalid evidence |

The current validator in [`assessment/blueprint.go`](../../apps/learning/backend/internal/learningpath/assessment/blueprint.go)
supports choice, gap fill, correction and transformation; it explicitly rejects audio
formats. The current curated `vocabulary_in_context` and `reading_comprehension` formats
still target **grammar** and cannot establish a separate vocabulary/reading result.
Skill activation requires a complete item source, validator, evaluator, evidence mapping,
result contract, UI and an end-to-end test for that skill.

### Generation, validation and rubrics

- A versioned `assess.items` request carries mode, section skill, band, language,
  explanation language, objective candidates, allowed formats, requested count and
  normalized previously asked prompts. The reply contains only slot-matched content,
  answer keys or rubric anchors, and optional rationale. Strict decoding rejects extra
  fields, wrong format/skill/band, invalid keys, duplicate text and references outside
  the sent candidates. The accepted question is snapshotted before issue. Existing
  Quick fallback is in [`commands/assessment_items.go`](../../apps/learning/backend/internal/learningpath/commands/assessment_items.go).
- Closed-answer evaluators use server-side keys; skip and unavailable evaluation are
  separate outcomes. Open writing/speaking replies use a versioned rubric with criterion
  IDs, score `[0,1]`, cited observations and feedback. The backend validates IDs and
  bounds values, computes weighted totals, and caps AI-origin evidence. Unsupported or
  failed rubric evaluation yields `evaluation_unavailable`, never a guessed pass.
- Writing criteria: task fulfillment, comprehensibility/coherence, language range and
  accuracy. Speaking criteria: task fulfillment, intelligibility, fluency and interaction
  only when the task actually contains interaction. Pronunciation has its own acoustic
  criterion; it is not inferred from speech-to-text. Criteria weights, anchors and
  thresholds must be versioned and calibrated before they affect level estimates.

## 3. Difficulty, gaps and stopping

The backend selects difficulty from the section's own independent evidence and prior;
an existing profile estimate may initialize a **bounded prior**, never become a new
assessment result by itself. The current Quick grammar policy is a versioned A1–B2
batch heuristic: six-question batches by default, 12 usable answers minimum and 30
submitted answers maximum. It is defined in
[`assessment/policy.go`](../../apps/learning/backend/internal/learningpath/assessment/policy.go).
That heuristic is not validated for vocabulary, reading or productive/audio skills.
Each new skill needs a tested adaptation policy, including answer format weights,
repeated/assisted exclusions, floor/ceiling behavior, uncertainty and workload cap.
The earlier Bayesian posterior in `ASSESSMENT_ENGINE.md` section 3 is a **proposal**, not
an approved implementation or a mandatory replacement for the verified heuristic.

Each section stops on its own sufficient coverage/confidence rule, workload cap, item
exhaustion, unavailable evaluator or learner request; it stores a reason. The overall
mode stops only according to its configured required sections and global budget. A
single successful batch, a skipped task or an AI judgement cannot establish a level.
The result distinguishes observed gap, suspected gap, sampled competence, conflicting
evidence and unknown/unassessed. Gap analysis must use objective-level evidence and
prerequisites where available; missing evidence alone is **unknown**, not a confirmed gap.

## 4. Evidence, learner profile and Learning Path

All modes append to the same [`learning_evidence`](../../apps/learning/backend/migrations/0036_learning_loop.sql)
ledger, with `source=assessment`, `practice_mode=assessment`, section/skill, objective,
question version, outcome, evidence mode, weight and assistance. One accepted attempt
produces at most one evidence row. Recognition weighs less than independent recall or
production. Audio or AI-scored evidence remains bounded until calibrated. Assessment
evidence updates skill estimates and recommendations but **does not by itself mark a
knowledge item mastered** or advance the review ladder; D-025/D-032 still govern mastery.
The existing Quick behavior writes each answer and its evidence in one transaction.

Results retain `assessmentType`, status, version, section results, coverage, uncertainty,
knowledge gaps, evidence IDs and suggested starting point. An assessment may refine
future path recommendations without deleting paths, activity history or older estimates.
The planner consumes only supported results and treats other skills as `unassessed`.
Completion is neither mastery nor certified CEFR proficiency. The current Quick
starting band is a conservative planning input, not a multi-skill CEFR certificate.

## 5. Storage and API rollout

Migration [`0039_assessment_sections.sql`](../../apps/learning/backend/migrations/0039_assessment_sections.sql)
adds `learning_assessment_sections` and `learning_assessment_section_questions`, plus
an evidence attribution view, without changing existing row shapes. Section rows store
the skill, order, lifecycle, policy and optional blueprint, progress and result. Unique
keys prevent duplicate skills/positions and multiple active sections for a test; question
links enforce matching test and skill. Answers and evidence remain in their existing
tables. Skill estimate history and multi-section commands remain future work.

The migration installs compatibility triggers before backfilling the existing Quick
attempts. Old writers that finish transactions while migration is running are serialized
by PostgreSQL DDL locks; later old and new writes mirror the parent test, question links
and answer counts. New Quick attempts save the full blueprint inside the existing policy
JSON as well as the section row, so old binaries can ignore the added field and results
survive rollback. Historical policies are copied exactly; missing historical blueprint
or section result stays NULL. Quick execution still uses the parent lifecycle under a
row lock, but loads its saved section policy and, when present, saved blueprint. The
implementation must continue to retain a compatible handler for every policy version;
a snapshot alone cannot preserve executable rules or a removed question bank.

**Rollout/rollback:** apply 0039 before deploying the new API, retain the bridge while
old instances can write, and monitor DDL lock duration on real data. The new API requires
0039: stop/roll back all new instances before rolling back the migration. Down removes
the mirrored section objects only if their contents are reconstructible from the legacy
tests and answers; it rejects independent or changed section data. Quick IDs are stable
across a safe down/up; Quick policies, snapshots, answers and results stay on the legacy
tables. Before future independent sections are written, plan and verify a data-preserving
rollback/export strategy; do not bypass the guard. 0037's own Down is destructive to
Quick attempts and assessment evidence and must not be used as a routine rollback.

The first read-only capability route is already implemented:
`GET /learning/path/assessments/capabilities?language=` (session-only), returning
only runnable mode/skill combinations. It currently returns grammar Quick for en/es;
the frontend has not switched its selector to this route. The existing Quick progress
response now includes one persisted section with an optional result. Extend start input
with `assessmentType` and, for skill mode only, `skill` when those modes are ready. Section selection,
pause/resume, history and comparison endpoints need explicit versioned contracts in
[`API_CONTRACTS.md`](API_CONTRACTS.md) before implementation. Authorization is scoped
to the authenticated user and profile. No route may advertise a mode/skill that cannot
complete its full workflow. The onboarding selector remains in
[`features/assessment/start.tsx`](../../apps/learning/frontend/src/features/assessment/start.tsx)
using shadcn/ui; enable a choice only from backend capability information for its
language, skill and required media features. Manual and scratch stay usable.

## 6. Capability dependencies and release gates

Existing audio support: backend TTS/cache
[`learningaudio`](../../apps/learning/backend/internal/learningaudio), exercise audio
routes in [`routes.go`](../../apps/learning/backend/internal/learningpath/routes.go),
client playback in [`audio-button.tsx`](../../apps/learning/frontend/src/features/learning/components/audio-button.tsx).
Practice uses browser `SpeechRecognition` in
[`speech.ts`](../../apps/learning/frontend/src/features/path/speech.ts); it yields only a
transcript and is insufficient for speaking fluency or pronunciation scoring. There is
no assessment microphone recording/upload/storage or audio-aware evaluator in this
repository today. Listening requires stable TTS playback and an accessibility decision;
speaking/pronunciation additionally require MediaRecorder capture, server upload,
storage/retention, consent, speech recognition and calibrated audio analysis.

Release each slice only after policy/unit tests, generator/validator fixtures, backend
integration including authorization/resume/idempotency, migration up/down/up where
applicable, and browser flow from onboarding through result and path effect. Record
the gate in `IMPLEMENTATION_STATUS.md`; otherwise status stays `IN_PROGRESS`.
