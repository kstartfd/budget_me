# Product vision

## 1. Statement

Learnings becomes an **adaptive language learning platform**: for any learner, at any
proficiency, in any supported language, it works out what they actually know, builds a
path from what they are missing towards what they want to do with the language, makes
them *retrieve and use* it, schedules what must be reviewed so it is retained, and
re-measures so the path keeps fitting them.

The central loop (the whole product is this loop):

```
Knowledge Base → Assessment → Learner Profile → Learning Path → Practice
      ▲                                                            │
      │                                                            ▼
 Adaptation ◄── Reassessment ◄── Review ◄──────────────────── Feedback
```

This is an **evolution of the existing product**, not a rewrite. Today's app already has
teachers per language, placement tests, library-based programs, nine exercise types,
timed lessons, a rule book, a spaced-repetition notebook and AI chats (`CURRENT_STATE.md`).
The vision keeps every one of those working and changes what is *underneath* them: the
facts the system stores and the decisions it makes.

## 2. Who it is for

| Learner | What the platform must do for them |
| --- | --- |
| Complete beginner | Start at the real beginning, never show a placement test they cannot read; explain in their own language |
| Intermediate | Find the real gaps behind "B1" rather than replay what they know |
| Advanced | Go beyond textbook grammar: tasks, production, nuance |
| Uneven skills (reads B2, speaks A2) | Separate estimates per skill; no single "level" hiding a weak skill |
| Goal-driven (work, travel, exam, relocation) | Paths and tasks shaped by the goal, not only by level |
| Multi-language | One independent profile per studied language, each with its own explanation language |

Supported axes: **studied language** x **explanation language** x **goal** x **available
time**. Today's content covers English and Spanish explained in Russian and English
(A1-B2); the architecture must not assume those.

## 3. Principles (they decide conflicts)

1. **CEFR is a reference scale, not the architecture.** Levels estimate proficiency and
   describe objectives; they never gate access, and "grammar completed" never equals "B1".
2. **Knowledge is a first-class, stable, versioned thing.** Content is separated from the
   generated programs that use it. Rules, words and goals are written and checked by
   people; the AI explains, exercises and evaluates, it does not author canon (existing
   principle in `curriculum.go`, kept).
3. **Evidence over completion.** A lesson being finished proves nothing. Mastery needs
   independent recall, application, and retention over time (`LEARNING_METHODOLOGY.md`).
4. **The backend owns the truth; AI assists.** State, authorization, identifiers, progress,
   prerequisites, assessment evidence, scheduling, mastery calculation: deterministic code.
   AI output is validated before it is stored and never edits authoritative state directly.
5. **Honest numbers.** No fabricated or decorative statistics; every figure traces to stored
   evidence; empty and incomplete data are shown as such (`PROGRESS_MODEL.md`).
6. **Never discard history silently.** Assessment and learning history are kept; deletion is
   an explicit, confirmed user action with a clear scope.
7. **The learner pays for AI.** AI is bring-your-own-key; features must degrade gracefully
   without an agent (everything deterministic keeps working) and must not multiply calls
   without the learner's knowledge.
8. **Incremental and reversible.** Every phase ships additive migrations, keeps old routes,
   and leaves the product working. No big-bang rewrite, no new infrastructure without need.

## 4. What the learner experiences when the vision is met

- **First open:** a short, adaptive check (minutes, not an hour) that ends as soon as the
  system is confident, can be abandoned half-way and still be useful, and can be skipped by
  stating a starting point.
- **Every day:** one screen answers "what should I do now?": a session of the length they
  chose, mixing new material, due reviews and a real communication task.
- **During practice:** answers are judged instantly where there is one right answer and by
  rubric where there is not; each mistake is explained and linked to the exact thing they
  need to relearn; repeated mistakes come back.
- **Over time:** an honest picture per skill — estimated level *and how sure we are* —
  plus what is retained, what is slipping, what to do next.
- **With the tutor:** conversation and explanations that know their goal, level, recent
  mistakes and the relevant rule — and what they say in conversation feeds the same
  evidence, validated.

## 5. Non-goals

- Microservices, a message broker, a new datastore, or a new frontend stack.
- Replacing shadcn/ui or the Next.js/Go stack.
- Owning content authorship by AI (no AI-written canonical grammar).
- Gamification as a substitute for evidence (streak badges etc. are fine as *views*, never as inputs).
- Replacing today's notebook/extension capture workflow; it stays, and later feeds the
  vocabulary model (D-010).
- Real-time multi-user features, marketplaces, payments (not in this plan).

## 6. Success measures (product-level, measured from stored evidence)

| Measure | Definition |
| --- | --- |
| Placement efficiency | median questions answered to reach a stop with confidence >= threshold |
| Placement validity | agreement between placement estimate and later practice performance (`PROGRESS_MODEL.md` 7) |
| Retention | share of reviewed items recalled at their due date |
| Time-to-useful-session | seconds from opening the app to the first exercise |
| Recurring-error half-life | sessions until a repeated error category stops recurring |
| AI cost per active hour | tokens per active study hour from `learning_agent_usage` |

Targets are set when the baselines exist (Phase 8); none are claimed today.

## 7. Relationship to other planned work

- **Roles, admin area, teacher viewer** (planned separately, not started): the vision's
  "teacher" is a *human viewer* of a learner's program via a read-only link. It is
  independent of the "AI teacher". Kept as a separate stream (`DECISIONS.md` D-012).
- **Frontend design references** arrive separately; this documentation fixes contracts, not
  layouts.
