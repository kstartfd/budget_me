# Assessment engine

Phase 3. Today's code lives in `learningpath/commands/{test,testgen}.go` and
`learningpath/placement`. Everything in section 3 onward is `PROPOSED` until Phase 3 starts.

## 1. Current implementation (facts)

```
POST /path/tests {language, translationLanguage, level?}
   └─ agent writes batch 1 (8 Qs, A1)  ── validateBatch ──► stored with answer keys
learner answers a batch (POST /tests/:id/answers)
   ├─ each answer graded by backend:  placement.Grade (normalised exact match)       [deterministic]
   └─ when the level's batch is full: agent "examiner" returns {verdict: climb|stop, level}
         guard rails (validateVerdict): climb needs >= 50% right; a perfect batch cannot stop;
         stop places at the batch level or one below.            [AI decides, backend bounds]
   ├─ climb  -> agent writes the next level's batch (A1 -> A2 -> ... -> C2), advance in one tx
   └─ stop   -> test done, level_reached = verdict.level
final test: same loop but one level (the program's), passed = verdict climb; pass archives the program
choose level: POST /path/levels creates a finished placement (no questions) up to the highest closed program level
```

Facts: 8 questions per level (final: 10); question skills are `grammar | vocabulary |
reading`; one open test per `(user, language, kind)`; questions never carry the key to the
client; answers are persisted per question; the result is one `level_reached` string and a
boolean `passed`; the program is built from `level_reached` plus the list of wrongly
answered questions (`Analyze`).

## 2. Problems against the plan

| # | Problem | Plan reference |
| --- | --- | --- |
| A-1 | The AI examiner decides placement; the backend only bounds it | section 8: backend owns assessment evidence |
| A-2 | Evidence = a boolean per question; questions are not linked to knowledge items or objectives | 6.3, Phase 3 |
| A-3 | Result is one level; no per-skill result, no confidence | "separate estimated proficiency from assessment confidence" |
| A-4 | Listening, speaking, writing are not assessed | 6.3 skill-specific |
| A-5 | Fixed 8 questions per level, always from A1 (unless a prior program) | "adaptive question selection", "evidence-based stopping" |
| A-6 | Abandoning a test discards its value; no incomplete result | "partial and incomplete assessments" |
| A-7 | Item quality unknowable: AI-written keys are never verified or analysed | validation before persistence |
| A-8 | No reassessment concept; no knowledge-item diagnostics | 6.3 |

## 3. Target design

**Principle:** the backend runs the measurement; the AI supplies *items* and *explanations*.
Nothing the AI returns can change a result except the bounded, validated scores of free-form
items (section 6).

### 3.1 Assessment kinds (one engine, parameters differ)

| Kind (`purpose`) | Goal | Skills | Stops when |
| --- | --- | --- | --- |
| `placement` | initial screening, builds the first path | grammar, vocabulary, reading (listening/speaking added when Phase 10 items exist) | every target skill reaches the stop rule, or budget |
| `skill` | measure one skill on demand | one | the skill's stop rule |
| `diagnostic` | find which *items* are missing around a target (knowledge-gap analysis) | one skill or one cluster | each probed item has a verdict or budget |
| `reassessment` | triggered re-measure (`LEARNING_METHODOLOGY.md` section 5) | the triggering skill(s) | as `skill` |
| `final` | confirm a program's level | the program's core skills | P(level >= program level) >= 0.8 or budget |

`manual start`: the learner states a level; it becomes the **prior** (peaked, not certain)
and the test then verifies it. Choosing a previously reached level without testing (exists
today, `POST /path/levels`) stays, recorded as `basis = manual`, confidence <= 0.2.

### 3.2 Measurement model — discrete Bayesian posterior per skill

State per `(assessment, skill)`: probability vector over 6 levels (A1..C2), `p[0..5]`.

- **Prior**: uniform; with a start point `ℓ0`: `0.6` on `ℓ0`, `0.2` on each neighbour (renormalised at the ends); with an existing profile estimate: its posterior, widened.
- **Likelihood** of a correct answer on an item of level `j` for a learner at level `ℓ`:
  `P(correct | ℓ, j) = g + (1 − g − s) · σ(κ·(ℓ − j + 0.5))`, with guess rate `g` by format
  (4-option choice 0.25, free text 0.05), slip `s = 0.05`, slope `κ = 2` (PROPOSED).
  Wrong answers use `1 − P`.
- **Update**: `p'[ℓ] ∝ p[ℓ] · likelihood`, normalised. Pure function, no I/O.
- **Weights**: evidence of weight `w < 1` tempers the update: `likelihood^w`.
- **Outputs**: level = posterior mode; `confidence = 1 − H(p)/log 6`; also `P(ℓ ≥ L)` for any `L`.

### 3.3 Adaptive item selection

For each remaining skill pick the one with the lowest confidence (ties: fixed skill order),
then pick the item level `j*` that maximises **expected information gain** over the 6
candidate levels (6 x 6 operations), then pick an unseen item at `j*` from the item source
(section 5). Never repeat an item or an already-asked prompt (exists today as `asked`).
Deterministic given the same state and item pool — required for reproducible tests.

### 3.4 Stopping rules (evaluated by backend after every answer)

For a skill, stop when **any** holds, after a minimum of 4 answers:

1. `max p >= 0.85` (PROPOSED `stop_confidence`);
2. answers on the skill `>= 12`;
3. the learner has been at both ends of the scale (a correct answer at the highest level offered and an incorrect at the lowest) — ceiling/floor reached.

Whole assessment stops when all target skills stopped, total answers `>= 30`, the learner
stops it (`incomplete`), or the AI agent becomes unavailable and the item pool for the
needed level is empty. The *reason* is stored (`stop_reason`: `confident`, `max_questions`,
`ceiling`, `floor`, `learner_stopped`, `no_items`, `agent_unavailable`).

### 3.5 Partial / incomplete results

An abandoned assessment is stored as `status = done` with `stop_reason = learner_stopped` (an *incomplete* result without a new enum value, see section 7 step 1); results are still
written for every skill with `>= 3` answers, with their (low) confidence. The path builder
accepts incomplete results: skills with confidence `>= 0.5` use the estimate, others plan
"from foundations of the lowest assessed level of the language" and say so in the
program's `summary`. Resuming continues the same posterior.

### 3.6 Outputs written (one transaction at completion; per answer for evidence)

- per answer: `learning_path_test_answers` (kept) **and** a `learning_evidence` row (`source_type=assessment`, `mode=recognition|production`, `item_id` when the item has one).
- per skill: a `learner_skill_estimates` row (`basis=assessment`) and an `assessment_results` row.
- legacy compatibility: `learning_path_tests.level_reached` = the **weakest core skill's level** (so existing program-building logic and old clients keep working) and `passed` as before.

### 3.7 Knowledge-gap analysis

After a `placement`/`diagnostic`, gaps are computed *by the backend*: for each skill, items
of levels `<= estimate + 1` that the learner got wrong or never saw, ranked by
`(1 − P(item known))` and by prerequisite depth (items whose prerequisites are also gaps
come later). This replaces the "list of wrong answers sent to the AI planner" in
`Analyze` with a deterministic ranked list the planner receives as data
(`LEARNING_ENGINE.md` section 2).

## 4. Role of the AI (and nothing more)

| Task | AI? | Safeguard |
| --- | --- | --- |
| Write assessment items | yes | schema validation, dedup, level tag, persisted with `generated_by='ai'`; key correctness monitored (section 5) |
| Verdict climb/stop, level | **no** (today yes) | replaced by section 3.4; kept in *shadow mode* during rollout (section 7) |
| Score free-form answers (write/translate/speak transcripts) | yes, rubric | closed rubric JSON, scores clamped 0..1, evidence weight <= 0.7 |
| Explain mistakes | yes | text only, stored with the attempt |
| Summarise results to the learner | yes (phrasing) | numbers come from the backend and are passed in; the AI may not alter them |

## 5. Item sources and quality

1. **Curated bank** (human-written, `kb`-linked, answer verified) — preferred; grows with the KB (`KNOWLEDGE_BASE.md` section 6).
2. **AI-written, validated, persisted** — today's behaviour, made safer: structural validation (exists), item level tag, `generated_by`, stored with stats.
3. **Item analysis**: for every item track exposures and correctness bucketed by the learner's *prior* posterior level. An item answered correctly by `< 40%` of learners whose prior level `>= j + 1` (after >= 8 exposures) is flagged `suspect` and excluded from selection until reviewed — this catches wrong answer keys without a second AI call.
4. Reuse across learners of AI-written items (cost saving, statistics) needs approval: **D-017**. Until then items are per-test, as today.

## 6. Free-form items

Writing / translation / speaking are scored against a **rubric** the backend sends (criteria,
CEFR descriptors for the target level) and the AI answers with a strict JSON score
(`AI_CONTRACTS.md`). The backend clamps and weights it. A deterministic *anchor check* is
required where possible (target form present, minimum length, language detected), and the
evidence weight is `<= 0.7` until rubric/human agreement is measured (Phase 10).

## 7. Migration and rollout (no big-bang)

0. **Frontend first (R-07):** release a frontend that tolerates unknown test `status` values (`incomplete`) and the new optional response fields, *before* the backend can emit them.
1. **Expand** (additive migration): new columns/tables (`DATA_MODEL.md` B.3); nothing reads them yet. `incomplete` is a new value of the Postgres enum `learning_path_test_status` (`ALTER TYPE … ADD VALUE`, irreversible) — or the status stays `in_progress`/`done` and incompleteness is a separate nullable column `stop_reason` (**recommended: avoids the enum change, D-009**).
2. **Backfill by replay**: replay every stored `learning_path_test_answers` through the posterior (item level = the question's `level`) to create `learning_evidence` and `learner_skill_estimates` (`basis=assessment`, flagged `legacy`). Idempotent; verify counts equal.
3. **Shadow mode**: new tests run the AI examiner *and* the backend rule; the backend rule's verdict is stored but the AI's still drives the test. Log disagreement rate and which side was later right (practice evidence).
4. **Switch** when agreement is acceptable (target: backend rule within one level of the AI in `>= 90%` of tests, and never above the AI's by more than one) behind a config flag; AI verdict kept for N releases as a monitor, then removed.
5. **Contract**: delete the verdict prompt and `validateVerdict` after one release without incident.

Old clients keep working throughout: endpoints and response fields are only *added*
(`progress`, `stopReason`, `skills`), see `API_CONTRACTS.md` section 5.

## 8. Acceptance criteria (Phase 3, summary — full list in `IMPLEMENTATION_PLAN.md`)

- Simulated learners (known level per skill, noisy) are placed within +-1 level in `>= 90%` of 1,000 runs, using `<= 20` answers on average.
- A learner who answers randomly never ends with `confidence >= 0.85` at a level above A2 (guess-rate sanity).
- Re-running the same answers yields the same result (determinism).
- Abandoning at any point yields a stored `incomplete` result and a resumable assessment.
- No code path lets an AI response change a level or a stop decision.
- Authorization: another user's assessment id is `404` on every endpoint.
