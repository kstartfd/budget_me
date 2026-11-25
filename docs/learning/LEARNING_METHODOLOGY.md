# Learning methodology

How each educational approach in the plan becomes a **concrete, testable system rule**.
Numbers marked `PROPOSED` are starting defaults to be tuned with real data (Phase 6/8); they
live in one Go config struct per engine so tuning never needs a migration.

## 1. Approach -> rule

| Approach | The rule the system enforces | Where it lives |
| --- | --- | --- |
| **CEFR** | CEFR labels *objectives* (`kb_objectives`) and *estimates* (`learner_skill_estimates`, per skill, with confidence). It never gates content and is never inferred from "topics completed". There is no stored single overall level; a UI summary is derived and labelled as such. | KB, `PROGRESS_MODEL.md` |
| **Mastery learning** | A knowledge item is `mastered` only when the evidence criteria in section 2 hold. A topic being completed changes the *path*, not mastery. Mastery can be **lost** (lapse -> `relearning`). | `LEARNING_ENGINE.md` |
| **Retrieval practice** | Evidence has a `mode`: `recognition` (pick), `recall` (produce a known form), `production` (free output). Recognition alone can never take an item past `learning`. New exercise sets must contain recall/production items for every item they teach. | exercises, evidence |
| **Spaced repetition** | Every item that has evidence gets a due time from its outcome history (section 3). Reviews are a first-class session component and a visible queue. | `learner_item_state.due_at` |
| **Task-based language learning** | A `communication_task` item = a realistic situation, a measurable outcome, a rubric. A task is evaluated on task success first, language accuracy second. Path generation includes tasks matching the learner's goal. | KB kind, tutor scenarios (exist today: restaurant, travel, shopping, interview) |
| **Communicative language learning** | Practice favours meaning-bearing production over isolated transformation; tutor conversation and written/spoken tasks are first-class evidence sources. | exercises, tutor |
| **Interleaved practice** | A session is not blocked by topic: new items, due reviews and applications are mixed, and items that `contrast` with each other are scheduled together (section 4). | `learningsession` |
| **Formative assessment** | Every attempt updates state; estimates are recomputed from evidence; a divergence between estimate and practice triggers reassessment (section 5). | assessment, adaptive |

## 2. Mastery (PROPOSED defaults)

States of an item for one learner: `unseen -> introduced -> learning -> practiced -> mastered`,
plus `relearning` after a lapse.

| Transition | Condition |
| --- | --- |
| `unseen -> introduced` | the learner was shown the item's explanation / completed its intro step (the *only* transition not based on a graded outcome) |
| `introduced -> learning` | first graded evidence exists |
| `learning -> practiced` | weighted recent score >= **0.8**, >= **3** pieces of evidence, of which >= **1** is `recall` or `production`, from >= **2** different days |
| `practiced -> mastered` | a successful `recall`/`production` at a **delay >= 7 days** after the previous success (retention shown), and weighted recent score >= **0.85** |
| any -> `relearning` | an `incorrect` outcome on a due review of a `practiced`/`mastered` item (a *lapse*, `lapses += 1`) |
| `relearning -> practiced` | two consecutive correct recalls on different days |

Evidence reliability `weight` (PROPOSED):

| Source / mode | Weight | Note |
| --- | --- | --- |
| deterministic graded `production` (cloze, order, fill, spoken match) | 1.0 | the system can verify it |
| deterministic graded `recognition` (choice, listen) | 0.4 | 25% guess rate |
| AI-rubric `production` (translate, write, tutor task) | <= 0.7 | capped until rubric agreement is measured |
| tutor-extracted (validated) | <= 0.5 | needs `item_id` that exists and belongs to the language |
| self-report | 0.1 | may set a *starting point* only, never mastery |
| each hint used | weight x 0.5 (floor 0.2); a correct answer with a hint is `partial` | |

Recency: weights decay with a half-life of **30 days** inside the "weighted recent score"
(PROPOSED); old evidence is never deleted, only down-weighted.

## 3. Review scheduling (PROPOSED)

Phase 6 starts from a **transparent stage ladder** generalising what exists
(`reviewDays = 1,2,4,7,15,30,60` days, `again`=back to stage 0 and due in 10 min):

| Outcome | Next |
| --- | --- |
| `correct` (recall/production) | stage + 1 (stage + 2 if latency fast and no hints) |
| `correct` (recognition only) | stage unchanged, due at the ladder interval x 0.5 |
| `partial` | stage unchanged, due at interval x 0.6 |
| `incorrect` | stage 0, `due_at = now + 10 min` (same session), then 1 day; `lapses += 1` |

Ladder-based now; a data-driven model (FSRS-style memory stability) is an *option once
>= 10,000 review events exist* (D-016). The scheduling policy is a pure function
`NextReview(state, outcome, mode, now) -> state'` with table-driven tests; the notebook's
existing `NextReview` is the reference implementation and **keeps working unchanged**.

The review queue is `SELECT ... FROM learner_item_state WHERE due_at <= now()` ordered by
overdue-ness x item importance; it never contains more than the session can hold.

## 4. Session composition (PROPOSED)

A session of `T` minutes is planned by time, not by item count:

| Slice | Share of time | Content |
| --- | --- | --- |
| Review | ~40% | due items, most overdue first, capped by `T` |
| New | ~35% | next path items whose prerequisites are `practiced`+; at most `daily_new` |
| Application | ~25% | a communication task, free production or tutor scenario using items from the other two slices |

Rules: no more than 2 consecutive exercises on the same item cluster; an item pair linked
`contrasts_with` is scheduled in the same session; a session never starts with the hardest
item; if there is no AI agent, application falls back to deterministic production kinds
(order, cloze, fill) and the session says so.

## 5. Formative assessment and reassessment triggers

The system re-measures when any holds (each is a testable predicate):

1. **Scheduled**: no assessment evidence for a skill in `N` active study days (PROPOSED 28), or the skill estimate's confidence decayed below 0.5.
2. **Divergence**: a skill's practice-derived estimate differs from its last assessed estimate by >= 1 CEFR band with confidence >= 0.6.
3. **Plateau/failure**: >= 3 lapses on items of one cluster within 14 days.
4. **Learner request** (always allowed).
5. **Path boundary**: the learner finishes a path block, or asks to move on.

Reassessment is a *short, targeted* assessment of the affected skill only (`ASSESSMENT_ENGINE.md`).

## 6. Content principles

- Explanations are short, example-led, in the learner's explanation language, with the target-language forms visible; theory stays human-authored (KB).
- Every exercise targets named knowledge items; an exercise without an item is not allowed after Phase 5.
- Errors are explained *and* linked to the item to relearn; the explanation is stored with the attempt.
- Feedback is immediate for deterministic grading; AI feedback may take a call and is clearly marked as generated.
- Difficulty is stepped within a level before stepping up a level; skill mix per path follows the existing library rule (every program contains vocabulary, reading/writing, listening and speaking where the library has them).

## 7. What this methodology deliberately does not claim

- It does not claim to *measure* CEFR exactly. Estimates carry confidence and the evidence behind them; they are advice, not certification.
- "Mastered" means "retained and used under the tested conditions", not "can never forget".
- Thresholds above are untested defaults. Phase 6 must ship with a simulation harness (synthetic learners with known forgetting curves) and Phase 8 with real-data calibration before thresholds are advertised to users.
