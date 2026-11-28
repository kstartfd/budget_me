# Quick grammar assessment — design handoff

Prototype: http://127.0.0.1:4173/?page=Assessments

Scope: the Quick grammar assessment for English and Spanish, from entry through the learning-path handoff. All questions, answers and results in this prototype are examples. A two-question sample is not a rule for calculating a real result.

## Interaction map and proposed copy

| State | Learner-facing copy | Actions and conditions |
| --- | --- | --- |
| Entry | **Quick grammar assessment**. “A short grammar check to help choose a starting point. You can pause and return later.” | **Start assessment** when no attempt exists; **Resume assessment** for an open attempt; **View result** for a finished attempt. English and Spanish only. |
| In progress | “Question [n] of [total]” and **Sample question** in the prototype. “Your submitted answers are saved.” | **Continue** submits the selected response; **I don't know** skips; **Save and exit** pauses. An unsubmitted selection is not described as saved. |
| Paused, no responses | **Assessment paused**. “No answers submitted yet. You can resume when you’re ready.” | **Resume assessment** returns to the first unanswered question. **Choose a starting point** offers self-reported level or A1. |
| Paused, responses recorded | **Assessment paused**. “Your progress is saved. Resume when you’re ready.” | **Resume assessment**, **View progress so far**, or **Choose a starting point**. Choosing a starting point does not imply the paused assessment is complete. |
| Partial, still resumable | **Assessment in progress**. “Your assessment is paused. There isn’t enough information for a starting point yet.” **Starting point: Not enough information**. Grammar: **In progress**. | **Resume assessment** returns to the saved attempt; **Choose a starting point** opens the manual/A1 choices. This is progress on a paused attempt, not a terminal partial result. |
| Finished, insufficient information | **Not enough information yet**. “There isn’t enough information to suggest a starting point. You can choose one yourself.” Grammar: **Not enough information**. | **Choose a starting point** opens the manual/A1 choices. **Back to Assessments** returns to the assessment entry. No resume action for an attempt that has ended. The prototype demonstrates this when both sample questions are skipped. |
| Finished grammar screening | **Grammar screening complete**. “This is a preliminary grammar check. It does not certify a level or demonstrate mastery.” **Starting point: Not shown in this sample**. Grammar: **Completed**. | **Continue to learning path** opens the confirmation step; **Back to Assessments** returns to the entry. Show an actual suggested starting point only when supplied and supported by the real result. |
| Path confirmation | **Create your [language] learning path**. “Your grammar answers will help shape your learning path. Other skills have not been assessed.” | **Create learning path** starts the existing path-creation transition; **Back** returns to the result. For manual/A1 choices, identify the chosen starting method instead. |

All result states show **Assessment coverage**. Grammar is the only assessed or in-progress skill. **Not assessed:** vocabulary, reading, listening, speaking, writing and pronunciation. Missing evidence is not a skill gap or an estimate of a level. The side note explains: “This check covers grammar only. Your learning path can include skills that haven’t been assessed.”

For the prototype's paused-to-manual path, the proposed confirmation copy says: “Your assessment remains paused. You can resume it later from Assessments.” This behavior must be confirmed before implementation (see decisions below).

## Another section in the future

Do not show **Complete another section** today: grammar is the only supported section, so returning to its completed row is a dead end. If an assessment later has another section that is available for the selected language and eligible to be started or resumed, show a secondary, specific action such as **Continue to [section]** or **Resume [section]**. It should open that section directly, preserve earlier results, and present accurate section status. Do not display it when no eligible unfinished section exists. The learning-path action remains available according to the product's agreed prerequisites.

## Responsive and accessibility notes

- Desktop: result and coverage appear in the primary card; the explanation sits alongside it. The primary action follows the starting-point summary.
- Telegram width: cards stack, text wraps, and actions wrap without horizontal scrolling. Controls have at least 44 px height in this flow. Long translations should wrap rather than truncate.
- A state change moves focus to the new heading. Keyboard focus is visible; answer choices use labeled radio controls, the open answer has a label, and status text does not depend on color alone.
- The interface is English copy intended to leave room for Russian and Spanish translations. Actual localization, screen-reader testing and validation inside Telegram remain implementation checks.

## Decisions needed before implementation

1. Can a learner create a path from a self-selected level while an assessment is paused, and can that same attempt be resumed after the path is created? The prototype proposes yes; confirm the existing eligibility and path-generation behavior.
2. For an ended partial attempt with some evidence, what result details may be shown, and when may the learner start a new attempt? A terminal partial result is distinct from a paused attempt and must not offer resume.
3. Which real result fields, confidence or uncertainty wording, and evidence conditions should produce a suggested starting point? Do not derive a level from the prototype's two sample questions.
4. Can Quick screening be started or repeated while an active program exists, and should the result affect that program or a future plan? The current backend restrictions need a product decision.
5. If an assessment later has multiple available sections, which sections are required before a learner may create a path, and in what order can they be completed?
