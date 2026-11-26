# Origin of the local design source

These four TSX files originated as the MagicPath **learning — Product Design** source mirror. They are now the [canonical editable local prototype](../README.md), imported directly by [preview/](../preview/). The `MagicPath/` directory name is retained for provenance; the hosted service is no longer required for design work.

- [LearningOverviewAssessmentCenter.tsx](LearningOverviewAssessmentCenter.tsx): shell, eight-section navigation, language selector, Overview states.
- [WorkspacePages.tsx](WorkspacePages.tsx): Learning Path, Knowledge Base, Practice, Progress, AI Tutor, Settings.

Overview and Learning Path expose the same nine design preview conditions through `Preview state`. The Path page renders no current program, unfinished assessment, active/completed program, current app snapshot, due review, insufficient data, missing AI key, and unavailable service. Both controls use the same local preview state, so navigation preserves the selected condition. No language is selected with the separate language switcher.
- [OnboardingPrototype.tsx](OnboardingPrototype.tsx): language setup and starting methods.
- [AssessmentPrototype.tsx](AssessmentPrototype.tsx): sample Quick grammar journey and disabled Full/Skill choices.

**Latest intended design:** these local files as of 2026-11-26. The former hosted [MagicPath project](https://www.magicpath.ai/files/476612556279123968) and [preview](https://designs.magicpath.ai/v1/mightily-shore-3385), revision `476638634322313216`, are historical references. The published preview predates the local Knowledge Base taxonomy redesign, assessment availability corrections, restart flow and additional Overview states. Its Knowledge Base still displays five flat example items. The local source instead shows 24 English grammar, 3 vocabulary and 2 functional-language sample items, category navigation and rule actions.

The original HTML design archives remain untouched in `docs/learning/frontend/`. The product's current backend capabilities are documented in [Navigation Handoff](../NAVIGATION_HANDOFF.md) and [Implementation Status](../../IMPLEMENTATION_STATUS.md). All prototype levels, answers, progress and insights are sample content. Do not infer implementation from a clickable prototype.
