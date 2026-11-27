# Origin of the local design source

These four TSX files originated as the MagicPath **learning — Product Design** source mirror. They are now the [canonical editable local prototype](../README.md), imported directly by [preview/](../preview/). The `MagicPath/` directory name is retained for provenance; the hosted service is no longer required for design work.

- [LearningOverviewAssessmentCenter.tsx](LearningOverviewAssessmentCenter.tsx): shell, eight-section navigation, language selector, Overview states.
- [WorkspacePages.tsx](WorkspacePages.tsx): Learning Path, Knowledge Base, Practice, Progress, AI Tutor, Settings.

All eight sections expose the same nine design preview conditions through a shared `Preview state`: active path, no program, completed path, current app snapshot, unfinished assessment, due review, insufficient data, missing AI key, and unavailable service. The selected condition persists across section navigation. No language is selected with the separate language switcher and opens the onboarding gate. See the [state matrix](../NAVIGATION_HANDOFF.md#preview-state-matrix) and [browser check](../preview/scripts/verify-states.mjs).
- [OnboardingPrototype.tsx](OnboardingPrototype.tsx): language setup and starting methods.
- [AssessmentPrototype.tsx](AssessmentPrototype.tsx): sample Quick grammar journey and disabled Full/Skill choices.

**Latest intended design:** these local files; use Git history to identify the revision. The former hosted [MagicPath project](https://www.magicpath.ai/files/476612556279123968) and [preview](https://designs.magicpath.ai/v1/mightily-shore-3385), revision `476638634322313216`, are historical references. The published preview predates the local Knowledge Base taxonomy redesign, assessment availability corrections, restart flow and nine-state coverage across the workspace. Its Knowledge Base still displays five flat example items. The local source instead shows 24 English grammar, 3 vocabulary and 2 functional-language sample items, category navigation and rule actions.

The original HTML design archives remain untouched in `docs/learning/frontend/`. The product's current backend capabilities are documented in [Navigation Handoff](../NAVIGATION_HANDOFF.md) and [Implementation Status](../../IMPLEMENTATION_STATUS.md). All prototype levels, answers, progress and insights are sample content. Do not infer implementation from a clickable prototype.
