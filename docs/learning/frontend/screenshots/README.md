# Learning design screenshots

Generated from the [local preview](../preview/README.md) with `npm run capture`. These are **viewport captures**, not full-page exports. Every learner name, statistic, CEFR label, skill estimate, assessment question/answer, insight, chat message, and Knowledge Base learner status shown in these images is **sample prototype content**. The screenshots do not prove that these capabilities exist in the application. The current implementation boundary is in [Navigation Handoff](../NAVIGATION_HANDOFF.md).

Desktop captures use **1440 × 900 CSS pixels**, device scale 1. Telegram-width captures use **390 × 844 CSS pixels**, device scale 1. The filename prefix is the capture order.

| State or screen | Desktop | Telegram width |
| --- | --- | --- |
| Overview: active plan | [01](01-overview-active-desktop.png) | — |
| Overview: new learner / language without plan | [02](02-overview-new-desktop.png) | [17](17-overview-new-telegram.png) |
| Overview: completed plan | [03](03-overview-completed-desktop.png) | — |
| Overview: unfinished assessment | [04](04-overview-unfinished-assessment-desktop.png) | — |
| Overview: due review | [05](05-overview-due-review-desktop.png) | — |
| Overview: insufficient data | [06](06-overview-insufficient-data-desktop.png) | — |
| Overview: missing AI key | [07](07-overview-missing-ai-key-desktop.png) | — |
| Overview: unavailable service | [08](08-overview-unavailable-service-desktop.png) | — |
| No language / onboarding | [09](09-onboarding-no-language-desktop.png) | — |
| Learning Path selected without a language (global onboarding gate) | [31](31-learning-path-no-language-desktop.png) | [32](32-learning-path-no-language-telegram.png) |
| Learning Path: active program, including Start a new plan | [10](10-learning-path-desktop.png) | — |
| Learning Path: no program | [21](21-learning-path-no-program-desktop.png) | [23](23-learning-path-no-program-telegram.png) |
| Learning Path: completed program | [22](22-learning-path-completed-desktop.png) | — |
| Learning Path: current app snapshot | [24](24-learning-path-current-desktop.png) | — |
| Learning Path: unfinished assessment | [25](25-learning-path-unfinished-assessment-desktop.png) | [30](30-learning-path-unfinished-assessment-telegram.png) |
| Learning Path: due review | [26](26-learning-path-due-review-desktop.png) | — |
| Learning Path: insufficient progress data | [27](27-learning-path-insufficient-data-desktop.png) | — |
| Learning Path: missing AI key | [28](28-learning-path-missing-ai-key-desktop.png) | — |
| Learning Path: unavailable service | [29](29-learning-path-unavailable-service-desktop.png) | — |
| Knowledge Base taxonomy | [11](11-knowledge-base-desktop.png) | [20](20-knowledge-base-telegram.png) |
| Practice | [12](12-practice-desktop.png) | — |
| Assessments: Quick grammar available; Full/Skill unavailable | [13](13-assessments-desktop.png) | [19](19-assessments-telegram.png) |
| Progress | [14](14-progress-desktop.png) | — |
| AI Tutor | [15](15-ai-tutor-desktop.png) | — |
| Settings | [16](16-settings-desktop.png) | — |
| Eight-section mobile navigation | — | [18](18-navigation-telegram.png) |

The capture script creates a fresh browser context for every image, selects the named sample state, waits for the UI transition, and records the local preview. Overview and Learning Path share the full `Preview state` list. Selecting a state on either page carries it to the other; the Path page renders its corresponding program, empty, assessment, review, insufficient-data, missing-key, or service-failure state. No language is selected through the language switcher and captured as onboarding. Regenerate after editing the source, then inspect desktop and mobile screenshots before committing them. In particular, inspect the Knowledge Base category tree and the Assessment availability labels, because those were stale in the former hosted preview.
