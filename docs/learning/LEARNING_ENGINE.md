# Learning engine

Covers the Learning Path Engine (Phase 4), the Adaptive Learning engine (Phase 6) and the
Study Session Engine (Phase 7). Section 1 is today's behaviour; sections 2-6 are `PROPOSED`.
Rules and thresholds referenced here are defined in `LEARNING_METHODOLOGY.md`; state and
evidence in `PROGRESS_MODEL.md`.

## 1. Current implementation (facts)

**Legacy program creation** (`BuildProgram` -> job -> `Analyze`):

1. Input: a finished *placement* test (level reached + every wrong answer) and the learner's survey (interests, variant, goal, hours/week).
2. Needs an AI agent (`400 "this needs an AI agent"` without one).
3. If the pair has a library and `reached <= B2`: the agent receives the library topics up to `reached + 1` (`libraryFor`) and returns `{summary, topics:[{id, why}]}`; backend validates (ids exist, 6-14 topics, unique, >= 2 non-grammar, listening+speaking present); topic text comes from the library.
4. Otherwise the agent writes the whole program (title/goal/theory/vocabulary per topic).
5. Saved in one transaction as a `learning_paths` row plus ordered `learning_path_topics`; exercise sets for the first `PathPrefetch` topics are then written in the background.

**Quick Screening:** the same job reads the persisted result and creates a canonical library plan
from gaps, recommended checks, documented prerequisites, goal tags and study hours. It adds
practice for unassessed skills and uses the existing path and review activity stores. The plan
can be created with the assessment evidence and library alone. A provider remains necessary
when exercises need to be generated. See [Quick Screening](QUICK_SCREENING.md).

**Progress inside a program:** a topic is `not_started -> in_progress` on the first answer
and `completed` when every exercise of its current set has a correct attempt
(`CompleteLearningPathTopicIfExercisesPassed`); the learner can also complete manually
(`POST /topics/:id/complete`). A final test confirms the level and archives the program.

**Not present:** prerequisites, per-skill levels, gap ranking, reviews inside a program,
program revision, session planning. Notebook SRS (`NextReview`) and daily sets are a
separate system for cards.

## 2. Path generation (target)

### 2.1 Inputs (all from the backend)

`profile` (goals, weekly minutes, session minutes), `skill estimates` (+confidence),
`item states`, `KB` (published items, relations, objectives), `gaps` (ranked list from
`ASSESSMENT_ENGINE.md` section 3.7), `due reviews`, `explanation language`.

### 2.2 Algorithm (deterministic core + optional AI shaping)

```
candidates  = published KB items of the learner's language
              with cefr_level <= max(skill estimate)+1  (per item.skill's own estimate +1)
              and state not in (practiced, mastered)
available   = candidates whose HARD prerequisites are practiced+ (or known by assessment with confidence >= 0.6)
              ── anything else is `blocked`: its missing prerequisites are pulled in first
priority(i) = w_gap·gap(i) + w_goal·goal(i) + w_unlock·unlock(i) + w_recur·recur(i) − w_cost·cost(i)
order       = topological order of the chosen set, ties broken by priority, units kept together
size        = fit to time: Σ estimated minutes ≤ weeks × weekly_minutes  (default 8-14 topics as today)
skill mix   = keep the library rule: grammar + vocabulary + reading/writing + listening + speaking
              where the KB has them
```

- `gap(i)` = `1 − P(known)` from the assessment posterior or item state; `goal(i)` = overlap of the item's categories/objectives with the learner's goal (domain tags such as `travel`, `work`, `exam`); `unlock(i)` = number of other candidate items whose only missing hard prerequisite is `i`; `recur(i)` = recent recurring-error weight (`PROGRESS_MODEL.md`); `cost(i)` = estimated minutes. Weights are one config struct (PROPOSED: gap 0.40, goal 0.25, unlock 0.15, recur 0.15, cost 0.05).
- **Prerequisite validation** is a pure function `ValidatePath(items, relations, states) -> []Violation`, run at creation, at every revision, and in tests against the real library. A path that violates a hard prerequisite is rejected, never saved.
- **AI shaping (optional):** the AI receives the already-validated ordered candidate list and may (a) write the one-sentence *why* per topic in the explanation language and (b) propose a swap within an equal-priority band. The backend re-validates; if the AI reply is unusable, the deterministic path is used. **Consequence: a path can be built with no AI agent at all**, removing today's hard dependency.
- **Without a KB** for the pair (today: any pair except the five library pairs), the current behaviour (agent writes the program) is kept as a clearly-labelled `source = generated` fallback until the KB covers the pair; topics carry no `kb_item_id`.

### 2.3 Goal-based activities

Goals (`profile.goals`: interests, domain, target skill emphasis) add `communication_task`
items matching the domain and weight vocabulary clusters by domain. A goal never bypasses
prerequisites; it only re-ranks within the available set.

### 2.4 Output

`learning_paths` + `learning_path_topics` as today, plus per topic `reason`
(`gap|goal|prerequisite|review|new`), `priority`, `kb_item_id`, and
`learning_path_topic_items` for multi-item units. The `GET /path` response only *adds*
fields (`API_CONTRACTS.md` section 5).

## 3. Dynamic path updates (Phase 4 core, Phase 6 triggers)

`AdjustPath(profile)` runs after: an assessment completes, a session ends, a topic
completes, or `>= 20` evidence rows accumulated since the last run. It is **idempotent and
bounded**:

| Operation | Condition | Guard |
| --- | --- | --- |
| Insert remedial item before X | recurring error or lapse cluster on an item that is a prerequisite of X, or `gap` high | at most **3** insertions per run |
| Defer item | a hard prerequisite fell to `relearning` | item stays in the path, marked `blocked` |
| Skip/drop item | item `mastered` by assessment with confidence >= 0.8 | marked `skipped` with reason, not deleted |
| Re-rank not-started items | priorities changed | never moves a topic that is `in_progress` or `completed` |
| Extend | fewer than 3 not-started topics remain | appends next ranked items |

Every run writes a `learning_path_revisions` row (`trigger`, `diff`) and the learner can see
**why the path changed** in one line. Completed topics, attempts and history are never
altered or removed by an adjustment.

## 4. Adaptive learning loop (Phase 6)

```
attempt / review / assessment answer
   └─ (same tx) evidence row ── UpdateItemState(evidence) ── state, mastery, due_at, lapses
                                  └─ if lapse or recurrence threshold: enqueue AdjustPath, add mistake journal row
priorities  = f(item states, estimates, goals)     (pure, recomputed on read; not stored)
review queue = items with due_at <= now             (ordered by overdue × importance)
```

- **Mistake tracking:** every `incorrect/partial` evidence with an `error_category` upserts a `learner_mistakes` row (journal). **Recurring error** = same `(item_id, error_category)` with `>= 3` incorrect evidence in 30 days and not `resolved`; resolution = 3 correct recalls of that item on different days after the last error.
- **Learning priority** (for sessions): `due-ness`, then `recurring`, then path order.
- The notebook SRS (`learning_entries`) is unchanged; a later phase may mirror cards as `phrase` items so a captured phrase gets the same state machine (D-010).

## 5. Study sessions (Phase 7)

Evolves `learning_path_lessons` (timer, pause/resume, `lesson_id` on attempts — all kept).

- **Duration** is a profile setting (`session_minutes`, default 30 = today's constant, bounds 5-120); `POST /lessons` accepts an optional `minutes`.
- **Plan**: `PlanSession(profile, minutes, now)` returns ordered `study_session_items` by the composition rules (review ~40%, new ~35%, application ~25%), interleaved, with item/exercise ids resolved. Exercises for chosen items are taken from existing sets or generated *before* the session starts (never mid-session blocking), falling back to deterministic kinds without an agent.
- **Resume**: session items carry `status` (`pending|done|skipped`); `GET /lessons/active` returns the plan and the cursor. A session left idle is closed lazily at its end as today.
- **Active time**: the client reports activity in heartbeats (<= 1 per 15 s); the server credits `min(delta, 30 s)` per heartbeat to `active_seconds` while not paused — so a forgotten open tab cannot inflate statistics. Wall-clock end and pause behaviour stay as today.
- **Persistence**: each item completion writes evidence (via the attempt path) and updates the item row in the same transaction; the end of a session triggers `AdjustPath`.
- Notebook "daily sets" remain a separate, simpler review mode and can later be offered as a session source.

## 6. Degradation matrix (no fake behaviour)

| Condition | Behaviour |
| --- | --- |
| No AI agent | path built deterministically; sessions use deterministic exercise kinds; written/translation items and tutor shown as "needs an agent" (existing settings link) |
| Agent down mid-session | session continues with the remaining deterministic items; failed items are `skipped` with reason, not marked wrong |
| KB missing for pair | AI-generated fallback program, clearly labelled; no item states, no prerequisite validation |
| Insufficient evidence | priorities use assessment/gap only; dashboard says "not enough data" rather than extrapolating |

## 7. Interfaces (Go sketch, for implementers)

```go
// pure, in learningreview / learningplan; no I/O, table-tested
func NextReview(s ItemState, e Evidence, cfg ReviewConfig, now time.Time) ItemState
func ApplyEvidence(s ItemState, ev []Evidence, cfg MasteryConfig, now time.Time) ItemState // rebuild path
func Priority(it Item, ctx PlanContext, cfg PriorityConfig) float64
func ValidatePath(items []Item, rel []Relation, st map[ItemID]ItemState) []Violation
func PlanSession(p Profile, minutes int, pool Pool, now time.Time) []SessionItem
```

Handlers (`commands/`) load state with sqlc, call these, and write results in one
transaction. The AI is never an argument to these functions.
