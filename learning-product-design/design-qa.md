# Design QA — Learning Path

## Comparison target and evidence

- Source: `docs/learning/frontend/Learning Path/Learning Dashboard technical specs.zip`, `Learning Path.dc.html`.
- Prototype: http://127.0.0.1:4173/?page=Learning%20Path
- Source and implementation were rendered together in the in-app browser using the temporary comparison page `/tmp/learning-path-ref/comparison.html`. The side-by-side browser capture is in the task's tool output; it was not saved as a separate image file.
- Desktop comparison: both embedded pages used a 1073 × 1261 CSS viewport, device pixel ratio 1; each was scaled equally to 60% in the combined capture.
- Mobile check: prototype rendered at 390 × 844 CSS pixels, device pixel ratio 1. The activity drawer and navigation were inspected in the browser.

## Comparison and findings

- **Typography:** both use the existing compact sans-serif hierarchy and muted supporting text. The prototype retains the local Geist/Inter/system stack.
- **Spacing and layout:** source and prototype show the objective, four summary cards, tabs, ordered modules and activity rows. At the comparison width they use one content column; the prototype keeps a second column on wider desktop screens and stacks cards on mobile.
- **Colors:** both use white cards, a neutral canvas, dark primary actions and blue, green and amber status labels. Labels remain readable without color.
- **Images and icons:** no content imagery is required on this screen. The prototype uses its existing Lucide icons.
- **Copy and content:** all five English and three Spanish sample modules and their activities were transcribed. Progress, dates and activity details are marked as sample content. The added restart and history copy implements the requested connected flow; the previous plan and results are represented as prototype data.
- **Expected differences:** the prototype places filters visibly in the Path header and adds restart/history actions and sample labels. It does not reproduce the source's speculative completion-time estimate. These are product requirements for this prototype.

## Corrections made during this review

1. The earlier transfer omitted two English modules, the Spanish path, and detail fields. Added the missing modules and source activity details.
2. The first detailed transfer did not mark the actual next step and filtered completed modules instead of completed activities. Corrected both behaviors and checked the activity drawer.
3. The original desktop breakpoint placed the side column beside the path too early. Adjusted the breakpoint and checked the source and revised prototype together at the same viewport.
4. Overview and Path pointed at different current activities. Connected the selected activity across Overview, Path and Practice.
5. Restored the source's ready, generating and failed path states and the four summary cards.

## Interaction and responsive checks

- All 12 preview states retained their selection across all eight sections; the no-language gate appeared in all eight sections.
- Checked the English path, Spanish path, activity detail drawer, completed and review tabs, filters, goal editing, A1 start, paused assessment, generation retry, missing-key and service-unavailable states. After creating a new sample plan, opened the previous plan in history and expanded its saved modules and activities.
- Checked the 390 × 844 mobile layout and opened an activity drawer. The document reported no horizontal overflow.
- The browser reported no console errors during the final capture.
- `npm run build` and `npm run test:sites` passed on 2026-11-27.

## Open product decisions and limits

- The scope and recovery behavior of the separate history/progress deletion action remain undecided.
- Assessment eligibility with an active program and server-owned goal persistence remain product decisions.
- This is a local, sample-data prototype; archived plans and current activity are represented in the browser session and do not change the learning application.
- This review covers the transferred Learning Path and its connected entry points; it is not a full pixel-by-pixel audit of the other seven sections.

final result: passed

## Quick grammar assessment result — 2026-11-27

- Compared the existing assessment entry, question, paused and result screens in the browser against the revised flow. The original result showed a fixed A2, named a vocabulary gap after a grammar-only screening, and offered **Complete another section**, which led back to an already completed grammar section. The Spanish example also displayed English questions.
- The revised result distinguishes completed grammar, paused progress and an ended attempt with insufficient information. It shows unassessed skills explicitly, omits an invented level and routes the learner toward either the saved assessment or a chosen starting point. The prototype uses sample questions and results; its two questions are not a real evidence threshold.
- Inspected desktop and 390 × 844 mobile layouts in the in-app browser. Result actions now precede the detailed coverage list so the primary action is visible sooner on mobile. The mobile document had no horizontal overflow. Browser captures are in the task's tool output; separate image files were not saved.
- Exercised an English completed result; Spanish sample questions; pause after one answer; partial progress; resume; completion; an all-skipped insufficient example; and the proposed path-from-manual flow followed by resuming the saved assessment. Confirmed the unfinished preview did not erase an existing completed sample.
- Used the keyboard to move focus to the primary result action and observed a visible focus outline. Answer choices and the free-text answer have labels. This was not a full screen-reader, contrast, translation or Telegram-client accessibility audit.
- `npm run build` and `npm run test:sites` passed. The path-after-pause behavior and the handling of real partial results remain product decisions, documented in `quick-assessment-handoff.md`.
