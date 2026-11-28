# Learning screen and interaction map

The [local design handoff](README.md) is authoritative for visual intent. This map links its controls to the existing application. For actual implementation status, read [Implementation Status](../IMPLEMENTATION_STATUS.md) and the cited code; the prototype itself uses sample data.

## Eight connected sections

| Section | Application destination | What it connects |
| --- | --- | --- |
| **Overview** | `/learning/overview` | Selected language's setup or plan state. **Continue learning** opens its current program topic; due review opens the tracked exercise; starting choices lead to assessment, claimed level, or A1. See [Overview UI](../../../apps/learning/frontend/src/features/home/components/overview-screen.tsx). |
| **Learning Path** | `/learning/path?language=…` | Active program → topics → exercises, plus plan history. **Start a new plan** archives the active program and returns to starting-method selection while keeping previous learning results. See [Path page](../../../apps/learning/frontend/src/app/learning/path/page.tsx). |
| **Knowledge Base** | `/learning/knowledge?language=…` | Published Rule Book → rule detail → free practice. The **personal notebook** is a separate collection at `/learning`; Rule Book reading does not establish mastery. See [Knowledge page](../../../apps/learning/frontend/src/app/learning/knowledge/page.tsx). |
| **Practice** | `/learning/practice?language=…` | Entry to due reviews, program activities, and free Rule Book/notebook practice. Only tracked answers feed saved progress. See [Practice page](../../../apps/learning/frontend/src/app/learning/practice/page.tsx). |
| **Assessments** | `/learning/assessment` | English/Spanish Quick **grammar** screening → preliminary result → program start; an unfinished Quick attempt can resume. Manual claimed level and A1 starts are alternatives. Full and Skill modes are unavailable. See [assessment screen](../../../apps/learning/frontend/src/features/assessment/screen.tsx). |
| **Progress** | `/learning/progress?language=…` | Stored topic, evidence, item and review totals for the selected language, with honest unknown/insufficient states. No fabricated all-skill CEFR estimates. See [Progress page](../../../apps/learning/frontend/src/app/learning/progress/page.tsx). |
| **AI Tutor** | `/learning/chat` | Existing topic chat for the selected language; a missing AI key leads to Settings. **Return to current activity** opens the program when one exists. Chat messages do not currently update mastery. |
| **Settings** | `/learning/settings` | Existing account and AI-provider controls; language management is at `/learning/languages`. |

[App shell](../../../apps/learning/frontend/src/components/app-shell.tsx) owns the sidebar and mobile menu. The language selector stores the active teacher/language and keys Path, Knowledge Base, Practice and Progress requests to it. Switching a language on those pages updates the route and data; no stale cross-language program or progress should remain. The interface locale is a separate Settings choice.

The prototype language selector changes the visible language context across all eight sections. Practice prompts and AI Tutor sample copy use the selected language; Progress and Knowledge Base do not carry an English learner estimate into another language. The catalogue itself remains sample reference content. The application must use language-keyed server data rather than copy any prototype values.

## Preview state matrix

The canonical `docs/learning/frontend/preview` source has nine `Preview state` choices in every section. On 2026-11-27 the separately running `learning-product-design` preview on port 4173 contained a later, untracked workspace source with **12** choices: it adds **Ready to create path**, **Creating learning path**, and **Path creation failed**. Its Assessment component differs from the canonical mirror only in the create-path button caption. The Assessment flow was inspected for P3-16; the three Path states have not yet been transferred to this version-controlled handoff. Do not assume that the nine-state matrix describes that running preview, or that the untracked copy replaces this directory's design authority.

Overview, Learning Path, Practice, Assessments, Progress, AI Tutor and Settings use the header selector; Knowledge Base places the same selector in its catalogue toolbar. The selection carries across navigation. **No language** is a separate global onboarding gate selected through the language switcher. The local [state check](preview/scripts/verify-states.mjs) opens all nine canonical states in all eight sections; representative [screenshots](screenshots/README.md) show the layout at desktop and Telegram width.

| State | Connected screen behavior in the local design |
| --- | --- |
| Active learning path | Overview and Path show a sample current program; Practice offers program, free and review entry points; Progress and Tutor use clearly labelled sample content. |
| New learner / no program | Overview offers Quick grammar, claimed level and A1; Path and Practice show no-program actions; Progress has unknown values; Rule Book stays browsable. |
| Plan completed | Overview and Path keep the completed program visible and offer extra practice or a new plan; history is retained in the design flow. |
| Current app snapshot | Shows an early sample program with limited metrics; no all-skill proficiency claim. |
| Unfinished assessment | Overview, Path and Practice route back to Assessments; its paused Quick grammar screen offers resume; no completed assessment result appears in Progress. |
| Review due | Overview, Path and Practice expose the due review entry; the queue and dates are sample. |
| Insufficient progress data | Program navigation remains available; Knowledge Base and Progress show unknown learner status and estimates. |
| Missing AI key | AI-dependent assessment, practice and tutor entry points show a Settings action; existing plan/history and static Rule Book remain visible. |
| Service unavailable | Live assessment, practice, progress and tutor actions show unavailable states; Path shows only a labelled last-known sample. |

Settings reflects the selected missing-key or unavailable condition. The no-language gate appears over every section until a language is selected. All numbers, levels, questions, answers, review dates, and chat messages in these states are illustrative; selecting a preview state does not simulate a backend response.

## Starting, returning, and restarting

- **No language:** onboarding creates a language profile with explanation language, goals, priorities and routine, then offers Quick grammar if supported, a claimed starting level, or A1 from scratch. Goals/routine in the current UI are stored in browser localStorage; this is not yet a server-owned learner profile.
- **Language without a program:** Overview repeats the three starting methods. Quick can pause and resume; completing it produces preliminary evidence, not mastery or certification.
- **Assessment question:** the application retains desktop navigation and the Telegram menu while an answer is open. Submitted answers persist; navigating away or pausing resumes at the next unanswered question. The result does not offer **Complete another section**, because grammar is the only available Quick section.
- **Active program:** Overview's continue action and Path return to the current topic. Due reviews open an existing tracked exercise; free practice is visibly separate.
- **Completed program:** Overview offers review/extra practice. Path retains completed topics and offers **Start a new plan**. The action archives the active program; prior programs and results remain in history. Deleting history/progress is a separate product action, not part of restart.

## State coverage and implementation boundary

| State | Current application | Prototype-only or incomplete design |
| --- | --- | --- |
| No language, language without program, active/completed program | Existing Overview/Path flows use application data. | Screenshots display sample names, program content and status. |
| Unfinished Quick assessment | English/Spanish grammar attempt can persist, pause and resume. | Prototype questions and answers are simulated in-browser. |
| Due review | Existing dashboard returns due activities; tracked attempts submit an `activityId`. | The example review queue and dates are sample content. |
| Insufficient progress data | Progress uses empty/unknown states and stored totals. | Seven-skill confidence bars and CEFR labels are design illustrations, not computed learner estimates. |
| Missing AI key / unavailable service | Existing pages can lead to Settings or show an error/retry state. | Overview's dedicated banners and all fallback copy are design states until matched to real backend errors. |
| Full/Skill Assessment; vocabulary/reading/listening/writing/speaking/pronunciation assessment | Not implemented end to end. | Display unavailable; do not unlock based on the prototype. |
| Daily plan, personalized insight, AI Tutor evidence | No complete persisted contracts. | Example metrics, insights, recommendations and chat content are sample only. |

The current app supports English, Russian and Spanish interface text. The prototype is English-language design copy; production copy belongs in the locale message files. See [Visual Specification](VISUAL_SPEC.md) for layout, responsive and accessibility rules.

## Open product decisions

1. Define the exact scope and recovery behavior of a separate **delete history and progress** action. The existing history deletion endpoint does not reset every learner record.
2. Decide whether **Start a new plan** pre-fills or clears saved goals and routine, and when those preferences move from browser storage to a server-owned profile.
3. Decide whether a learner can repeat Quick screening while an active program exists; the current backend rejects a new placement with an active program.
4. Define if and when Rule Book items and personal notebook entries are linked. Their storage and review flows are separate today.
5. Define backend contracts and evidence thresholds before implementing skill estimates, daily sessions, learning insights, AI Tutor evidence or audio-based assessment.
