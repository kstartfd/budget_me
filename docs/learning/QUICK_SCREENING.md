# Quick Screening

**Scope note (2026-11-26):** This file specifies the implemented grammar-only
milestone. The target Quick mode additionally includes vocabulary and reading through
the shared engine. Full and individual Skill Assessment are separate target modes;
see [ASSESSMENT_MODES.md](ASSESSMENT_MODES.md). None is implied by this milestone.

Quick Screening extends the first learning loop after its V1 production smoke check. It
produces a preliminary grammar profile, gaps and a starting plan for English and Spanish.
The difficulty labels describe the library material sampled. They are provisional,
uncalibrated product heuristics and confer no CEFR certification.

## Existing flow and compatibility

Onboarding previously used level placement under `/learning/path/tests`. The agent wrote
an eight-question set, the server graded it, then the agent decided to climb or place the
learner. Program generation used the single level and a list of missed answers. Questions,
attempts, resumable tests, teacher profiles and build jobs already exist and are reused.
Historical generated questions have no reliable knowledge-item reference. They remain
readable as legacy test history, and the evidence ledger starts with attributed attempts.

Quick Screening is an explicit placement purpose used by the new assessment entry point.
The API keeps its legacy `kind` and `status` fields. Existing placements without a purpose,
final tests and manual levels retain their behavior. Starting from scratch still uses A1 as
a manual starting choice. The other assessment purposes are reserved for future blueprints.

## Blueprint `grammar-quick-v1`

| Dimension | Definition |
| --- | --- |
| Learning language | English or Spanish. Question content is in the learning language. Objectives resolve in the profile’s Rule Book locale, with learning-language library fallback for unsupported pairs. |
| Skill | Grammar, including recognition and controlled production. Vocabulary, reading, writing, listening, speaking and pronunciation remain explicitly unassessed as separate skills. |
| Knowledge categories | Verb forms, noun phrases, sentence structure and clause linking. Each A1–B2 band covers three categories. |
| Canonical knowledge and learning objective | Every target comes from an existing `curriculum.Topic` with a Rule Book entry. `learning_knowledge_items(language, key)` stores that existing key (legacy name `kb_items` remains a view), and the question snapshots its existing learning goal and category. The bank makes the full mapping reviewable. |
| Difficulty | A1, A2, B1 and B2 library bands. An optional self-reported band sets the first batch. Default A2 lets the learner move in either direction. |
| Formats | Multiple choice, open gap fill, marked error correction and controlled sentence transformation. Vocabulary in context and short reading comprehension are recognition formats when they probe a specific grammar objective. |
| Minimum evidence | 12 independent, usable graded answers, at least three categories and a replicated or bracketed difficulty boundary. The minimum requires multiple complete batches. |
| Maximum workload | 30 submitted answers, including skips and unavailable evaluations. A shorter final batch is issued when necessary. No time limit runs while a learner is paused. |
| Stopping policy | Finish after satisfying the evidence criteria. Otherwise stop at the workload cap, exhaustion of validated questions, or the learner’s request. Return partial results with the available evidence. |

The bank in `internal/learningpath/assessment/bank.txt` contains 48 original questions
per language, with six targets per band and two different tasks per target. These are
project-authored probes. The fixture tests check every reference and prerequisite against
the Rule Book. No textbook exercises were imported.

The backend selects the band, skill, item, learning objective and format. It balances
categories and favors eligible unseen bank questions. A learner is never served the same
normalized prompt and context twice within their assessment history. If the bank cannot
fill a batch, the existing agent integration generates the missing slots. Its input contains
the backend-selected targets. Its JSON response contains content and a bounded key, and
rejects unknown fields, duplicate prompts, invalid formats or malformed keys. Generated
items are scoped to the current test under D-017. Question ID and opaque version ID are
persisted with the accepted content. Choice order is shuffled on issue.

## Evaluation and adaptation

- Grading compares choice values and constrained answers against server-side keys.
  Controlled open forms accept listed alternatives with Unicode normalization, case,
  whitespace and equivalent apostrophes. The key can specify a bounded set of curated
  partial-credit responses. The learner can skip, and an unusable evaluation is tracked
  separately. No score, key or answer feedback is returned during the assessment.
- Independent graded answers are usable. Assisted, repeat, skipped and unavailable
  attempts carry zero evidence weight. Partial responses carry a 0.5 score, and only
  correct or incorrect responses vote toward a difficulty adjustment. Recognition weighs
  0.4 and recall or production 1. Generated questions use 70% of the corresponding weight.
- For a fully submitted six-question batch, at least five correct votes favor moving up.
  At least four incorrect votes favor moving down. Three or four correct with the rest
  incorrect holds the band for more evidence. A vote requires at least two represented
  categories and at least two decisive curated responses per batch. For other sizes
  `n >= 3`, upward requires `ceil(5n/6)` correct and downward `ceil(4n/6)` incorrect.
  A batch of one or two can contribute item evidence and cannot change the band.
- A boundary requires positive evidence in one band and negative evidence in the next band,
  or two net directional batches at the A1 floor or B2 ceiling. Split responses at a band
  or stronger results above a weaker band hold the assessment open for further evidence.
  A single successful batch never places a learner.
- The policy is snapshotted per assessment. New sessions read `ASSESSMENT_BATCH_SIZE`
  (3–6), `ASSESSMENT_MIN_QUESTIONS` (at least twice the batch size) and
  `ASSESSMENT_MAX_QUESTIONS` (between the minimum and 60). Defaults are 6, 12 and 30.

Each submission writes the existing answer row and a single append-only evidence row in one
transaction. The attempt identifies its immutable question version, objective, canonical item,
skill and band. The ledger records evidence mode, score, weight and assistance under
`source=assessment`, `practice_mode=assessment`. D-025 and D-026 keep assessment evidence,
topic practice, review activities and mastery distinct. Assessment evidence informs a
profile estimate and recommendations. It does not directly advance mastery or the review ladder.

The row lock serializes submission, pause, completion and cancellation. Posting the same
accepted answer again returns the current state. Changing a recorded answer returns a conflict.
New-batch preparation runs after commit, so disconnects and provider failures leave accepted
answers intact. If a request is interrupted, resume continues from the durable state. If the
provider fails and no valid batch can be prepared, the server persists the partial result.
An initial failure with no attempt is `failed`. Cancelling preserves the attempt history.

## Outcomes and planning

Two distinct usable correct attempts, including recall or production and curated confirmation,
can demonstrate a sampled objective. Two independent misses with curated confirmation and no
contrary signal form a strong gap. A lone miss or misses lacking curated confirmation form a
suspected gap. Mixed, skipped and low-quality observations need more evidence. Unseen
items are `not_assessed`. A successful higher-band answer supplies no evidence about
a lower-band item or prerequisite.

The results record references to the exact evidence rows, an optional grammar difficulty
range, sufficiency, strengths, gaps and proposed checks and objectives. The program build
uses gaps and checks, a small explicit prerequisite map, the existing canonical library,
study time and goal tags. It leaves room for practice in unassessed skills and continues to
use the existing typed practice and due-review activities. Legacy `levelReached` is only
a path scheduling label. It uses the lower bound or A1 when the range is unknown. An archived screening plan
counts toward legacy level selection after its final test is passed.

Generated keys receive structural validation and reduced weight. Full semantic rubrics,
human agreement checks, cross-learner generation reuse, item calibration, broader
prerequisite coverage and additional assessment modes are later work. The constrained
open responses in this blueprint use finite answer keys. Provider evaluation can be added
for future semantic formats through the existing agent integration, using the persisted
outcomes `partial` and `evaluation_unavailable`.

Migration 0039 adds the section table and a question association, backfills one grammar
section per Quick test and mirrors parent/answer writes so old binaries continue to
work. New attempts retain their section blueprint inside the existing policy JSON and
read the section's persisted policy when resuming. Historical tests that recorded only
the policy keep their missing blueprint or section result unknown. Results and policy
remain on the legacy test for compatibility. See the [storage and rollback procedure](ASSESSMENT_MODES.md#5-storage-and-api-rollout).

P3-16 routes allocation, generation validation, evaluation, decisions and aggregation
through the shared `SectionEngine`. The command layer retains the transaction/parent
row lock and 0039's legacy bridge. Assessment UI uses the persisted grammar section when
present and the older assessment fields otherwise. The result labels suspected gaps
as needing another check and does not offer a nonexistent next skill section.

Migration 0037 adds fields and constraints to the existing tables. Its Down step
removes the new assessment sessions and evidence. Legacy tests, answers and program
history survive the round trip.

Implementation locations: `learningpath/assessment`, `learningpath/commands/assessment*.go`,
`queries/learning_assessment.sql` and migrations `0037_adaptive_assessment.sql`,
`0039_assessment_sections.sql`.
