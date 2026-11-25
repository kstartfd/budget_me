# Vertical slice V1 — the first complete learning loop

Decision: D-024 (accepted 2026-11-25). Authority to implement: D-002 (accepted, delegated).
Scope and vocabulary: D-025, D-026 (accepted with the owner's revisions), D-030..D-033.

**Goal.** Close the whole loop once, thin, on production data, before any horizontal phase is
finished. Every table, function and endpoint here is a *subset* of what P1, P2, P5, P6, P7, P8
need, so nothing is thrown away.

```
library rule (grammar topic of the learner's program)
  -> exercise (the existing set of that topic)            [exists]
  -> answer                                               [exists]
  -> check (deterministic / the learner's agent)          [exists]
  -> attempt stored                                       [exists]
  -> EVIDENCE   append-only row, same transaction         [V1]
  -> KNOWLEDGE STATE of the item updated                  [V1]
  -> REVIEW scheduled (due_at on the ladder)              [V1]
  -> minimal DASHBOARD + typed ACTIVITIES (practice, review)  [V1]
  -> a due review is answered -> evidence -> state -> next review   [V1]
```

## 1. Vocabulary (owner, D-026)

| Concept | What it is | In the code after V1 |
| --- | --- | --- |
| Knowledge Item | the canonical piece of knowledge | `kb_items` row `(language, key)`, created from the library on first use; the key is the library topic id |
| Learning Path Topic | a topic in the learner's personal program | `learning_path_topics` (unchanged); `source_id` = item key |
| Learning Activity | one concrete learning action with a type, goal, linked knowledge, status, parameters | `learning_activities` (new) |
| Exercise | one concrete exercise | `learning_path_exercises` (unchanged) |
| Study Session | a container of chosen activities | **not in V1**; the existing lesson timer stays as it is |
| Attempt | one try at an exercise | `learning_path_attempts` (unchanged shape) |
| Evidence | proof of what the learner showed | `learning_evidence` (new, append-only) |

Activity types (closed): `learn`, `practice`, `review`, `apply`, `assess`. V1 produces
`practice` and `review` only. Completion is stored on the activity and does not depend on any
session.

## 2. Source and practice mode (owner, D-025)

The **source** of an exercise and the **mode** of practice are independent.

| Column | Values | V1 |
| --- | --- | --- |
| `source` (where the exercise came from) | `rulebook`, `learning_path`, `study_session`, `review`, `ai_tutor`, `assessment` | writes `learning_path` and `review` |
| `practice_mode` | `free` (D-107 behaviour: no stored history, no evidence, no mastery, no path or schedule effect), `tracked`, `assessment` | stores `tracked` only; `free` never produces a row; `assessment` reserved |

Rule-book practice stays exactly as it is (free). Making it *tracked* later is a new
`source=rulebook, practice_mode=tracked` path that writes the same evidence; nothing in the
schema has to change for it.

## 3. Data (migration `0036_learning_loop.sql`, expand-only, with Down)

```sql
ALTER TABLE learning_teachers ADD CONSTRAINT learning_teachers_id_user UNIQUE (id, user_id);

kb_items(id, language, key, kind CHECK ('grammar','topic'), skill, cefr_level, unit, created_at,
         UNIQUE (language, key))                      -- subset of DATA_MODEL B.1; CHECK widened in P1

learning_evidence(id, user_id, teacher_id, item_id -> kb_items, skill,
                  source, practice_mode, ref_type ('attempt'), ref_id,
                  mode ('recognition','recall','production'), outcome ('correct','incorrect'),
                  score, weight, assisted bool, attempt_no, hints_used, occurred_at,
                  UNIQUE (ref_type, ref_id),           -- idempotent: one evidence per attempt
                  FOREIGN KEY (teacher_id, user_id) -> learning_teachers ON DELETE CASCADE)

learner_item_state(teacher_id, user_id, item_id, state, score, weighted_sum, weight_sum,
                   evidence_count, days, recall_successes, streak, streak_day, stage, lapses,
                   last_evidence_at, last_success_at, due_at, updated_at,
                   PRIMARY KEY (teacher_id, item_id))   -- derived, recomputable from evidence

learning_activities(id, user_id, teacher_id, type, status ('open','completed','skipped'),
                    reason, goal, item_id, topic_id, params jsonb, created_at, completed_at,
                    partial UNIQUE: one open review per (teacher, item), one open practice per topic)
```

Wire values are lower-case snake (D-027). Review state (`not_scheduled/scheduled/due/overdue`)
is derived from `due_at` at read time and never stored (D-028).

## 4. Rules (all PROPOSED defaults; one table in `learningprofile`, tested, replaceable)

**Evidence from an attempt.** Exercise kind -> mode and weight:

| Kind | Mode | Weight | Why |
| --- | --- | --- | --- |
| `choice`, `listen` | recognition | 0.4 | one right option of a few |
| `fill` | recall | 1.0 | the form is produced, checked against the key |
| `cloze`, `order`, `speak` | production | 1.0 | produced, checked against the key / the phrase |
| `translate`, `write` | production | 0.7 | judged by the learner's agent; capped until agreement is measured |
| `read` | recall | 0.7 | judged by the learner's agent |

A retry (`attempt_no` > 1) counts half (floor 0.2). Program attempts are `assisted = true`:
the exercise shows its key before the first answer (G-07). Review attempts are
`assisted = false`: the review exercise never carries its key.

**Skill.** `listen` -> listening, `speak` -> speaking, `read` -> reading, `write` -> writing,
otherwise the topic's own skill.

**Knowledge state** (`unknown` = no row; D-028):

| Transition | Condition |
| --- | --- |
| no row -> `learning` | first graded evidence |
| `learning` -> `practiced` | weighted recent score >= 0.8, >= 3 evidence, >= 1 correct recall/production, evidence on >= 2 different days |
| `practiced` -> `mastered` | an **independent** (not assisted) correct recall/production, >= 7 days after the previous success, weighted recent score >= 0.85 |
| `practiced`/`mastered` -> `relearning` | an incorrect answer to a due review (a lapse) |
| `relearning` -> `practiced` | two correct recall/production in a row on different days |

Weighted recent score: weights decay with a 30-day half-life. A single recognition success can
never take an item past `learning`. Mastery is a **state**, not a probability; nothing in V1
shows a percentage or a CEFR level.

**Review schedule** (ladder 1, 3, 7, 14, 30 days, D-029). `stage` counts consecutive
successes. A correct answer advances the stage **only when the item is due** (practising early
or answering six exercises in a row advances nothing), a recognition success never goes past
stage 1, an incorrect answer resets the stage to 0 and schedules the item for tomorrow.

**Attribution.** Only a topic with a library `source_id` has an item. A topic written by the
agent alone (no library for that pair) produces **no** evidence in V1 (documented limitation).

## 5. API (session-only, existing conventions, additive)

| Endpoint | Purpose |
| --- | --- |
| `GET /learning/path/activities?language=` | planned activities: open `review` activities for due items (most overdue first, at most 5, each with its exercise **without the key**) and the `practice` activity of the current topic. Materialises rows idempotently; a finished practice activity is closed here |
| `POST /learning/path/exercises/:id/attempt` | unchanged; optional body field `activityId` makes it a review attempt: allowed on a completed topic, key never required, evidence `source=review`, the activity is completed in the same transaction |
| `GET /learning/path/dashboard?language=` | minimal dashboard: counts by knowledge state, reviews due/overdue/upcoming and the next due time, evidence totals, the planned activities |

Errors: unknown or foreign id -> 404, not due / already completed -> 409, bad body -> 400.
Responses keep the existing envelope. No existing response changes shape, so the closed `z.enum`
hazard (R-07) does not apply.

## 6. Code map

| Piece | Place |
| --- | --- |
| Pure rules (mode/weight, state machine, ladder) | `internal/learningprofile` (new, no I/O) |
| Evidence + state + schedule in the attempt transaction | `internal/learningpath/commands/loop.go`, hooked into `attempt.go` |
| Activities and dashboard reads | `internal/learningpath/queries/loop.go` |
| SQL | `queries/learning_loop.sql` -> `internal/db` (sqlc 1.31.1) |
| HTTP | `learningpath/controller_loop.go` + routes |

## 7. Tests (written with the code)

| Layer | Cases |
| --- | --- |
| Pure | mode/weight table; transitions (practiced, mastered needs independent + 7 days, lapse, relearning); ladder; early practice does not advance; recognition cap; retry weight; invariants over random sequences (score in [0,1], stage in range, due in the future) |
| Migration | up -> down -> up with seeded legacy rows; existing data untouched |
| DB | idempotent evidence (second insert of the same attempt is a no-op), composite FK cascade on teacher deletion, partial unique indexes |
| HTTP, the loop | program -> exercise -> attempt -> one evidence row, state `learning`, `due_at` tomorrow, dashboard shows it; same-transaction rollback (failure after the attempt insert leaves nothing); no evidence for a topic without a library key; wrong answer lapse path |
| HTTP, review | due item -> activity with exercise without the key -> review attempt on a completed topic -> state, next due, activity completed; not-due -> 409; second answer to a completed activity -> 409 |
| Security | other user's activity/exercise -> 404; extension token -> 401; no token -> 401 |
| Regression | the whole existing suite unchanged (262 pass baseline) |

## 8. Out of V1 (deliberately)

Assessment engine changes, path engine changes, prerequisites, KB content tables and the read
switch (`kb_item_versions`, relations, search), profile endpoints and goals, study sessions,
mistake journal, AI insights, hints and answer reveal, the `outcome` column with `needs_review`
and `unavailable`, idempotency keys for attempts, lexemes, rule-book tracked practice,
assessment-mode evidence, replay of past attempts into evidence (P2-05), and any frontend
change. The frontend design arrives separately; V1 exposes the data it will need.

## 9. Known limitations to carry

1. Attempts made before V1 produce no evidence (no replay yet): the ledger starts at deploy.
2. Evidence from guided program practice is `assisted`; only an independent review can master an item.
3. An item with no stored exercise (history cleared) is not offered for review.
4. Two taps on "check" still store two attempts (A-6 is later); each gets its own evidence row.
5. A failed explanation call still aborts the attempt as before (A-5 is later).
6. Day boundaries are UTC.
