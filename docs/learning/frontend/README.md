# Learning design handoff

This directory is the **canonical, version-controlled design location** for the Learning app. Read this page first. The hosted MagicPath service is no longer part of the design workflow.

## Authority and reading order

1. [Design source](MagicPath/LearningOverviewAssessmentCenter.tsx), [workspace pages](MagicPath/WorkspacePages.tsx), [onboarding](MagicPath/OnboardingPrototype.tsx), and [assessment](MagicPath/AssessmentPrototype.tsx) are the **latest intended prototype**. The `MagicPath/` directory name records their origin; these files now run locally and are the single editable UI source.
2. [Visual specification](VISUAL_SPEC.md) explains the shared layout and interaction rules. Edit it when a visual rule changes.
3. [Navigation handoff](NAVIGATION_HANDOFF.md) maps prototype actions to application routes and marks implementation boundaries and unresolved decisions. Check [Implementation Status](../IMPLEMENTATION_STATUS.md) and the application code for actual behavior.
4. [Local screenshots](screenshots/README.md) are generated evidence of the source at fixed viewports. Regenerate them after a design change; they are not an independent design authority.
5. The ZIP archives below are **original historical references**. Keep them unchanged. Use them to recover intent when the latest source lacks detail, especially Knowledge Base taxonomy.

The latest intended revision is this repository's **local TSX mirror as of 2026-11-26**. It includes a connected eight-section workspace, new/completed/exception Overview states, a revised Knowledge Base with 24 English grammar, 3 vocabulary and 2 functional-language sample items, and honest Quick grammar availability. The last published [MagicPath preview](https://designs.magicpath.ai/v1/mightily-shore-3385), revision `476638634322313216`, predates these changes. In particular, its Knowledge Base still shows five flat sample items. Do not use that hosted preview for new implementation decisions. The local source and screenshot set are the reviewable revision; record a Git commit hash here after this handoff is committed.

## Run without MagicPath

Node.js 20.19+ and npm are required. From the repository root:

```bash
cd docs/learning/frontend/preview
npm ci
npm run dev
```

Open `http://127.0.0.1:5187/`. The server is local only. To rebuild the version-controlled screenshots in a second terminal:

```bash
cd docs/learning/frontend/preview
CHROME_PATH=/usr/bin/google-chrome npm run capture
```

`CHROME_PATH` may point to another installed Chromium/Chrome executable. If Playwright's bundled Chromium is installed, omit it. The capture script uses the running local server and writes `screenshots/*.png`. `npm run build` verifies that the editable source compiles. All direct dependencies are exact versions in [package.json](preview/package.json) and the transitive tree is pinned by [package-lock.json](preview/package-lock.json). No hosted design service or backend is needed. See [preview README](preview/README.md) for isolated checkout verification.

## Existing material inventory

| Material | Role and state |
| --- | --- |
| `MagicPath/*.tsx` | Local source mirror; latest intended design. The app preview imports these files directly, with no second copy. |
| [MagicPath/README.md](MagicPath/README.md) | Origin and publication history; retained for provenance. |
| [MagicPath/design-qa.md](MagicPath/design-qa.md) | Knowledge Base visual QA history; the hosted run was blocked before local preview existed. |
| [NAVIGATION_HANDOFF.md](NAVIGATION_HANDOFF.md) | Route/interaction map and capability boundary; maintained for coding agents. |
| [Learning Dashboard archive](Learning%20Dashboard/Learning%20Dashboard%20technical%20specs.zip) | Original HTML design references, including `Knowledge Base.dc.html`; retained verbatim. |
| [Dashboard Onboarding archive](Dashboard%20Onboarding/Learning%20Dashboard%20technical%20specs.zip) | Original onboarding HTML references; retained verbatim. |
| [Learning Path archive](Learning%20Path/Learning%20Dashboard%20technical%20specs.zip) | Original path and canvas HTML references; retained verbatim. |
| [Assessment archive](Assesment/assesment.zip) | Original assessment HTML references; directory spelling is historical. |

The archives contain standalone `.dc.html` pages, `support.js`, and the originally embedded image. The two archives named `Learning Dashboard technical specs.zip` are different files with different contents; neither was replaced. The original Knowledge Base screenshot provided in the design conversation informed the local taxonomy source, but the conversation attachment itself is not versioned in this repository. Its closest durable reference is `Knowledge Base.dc.html` in the Learning Dashboard archive.

## Change workflow

Edit the relevant `MagicPath/*.tsx` file, run `npm run build`, inspect the local preview at desktop and Telegram widths, then run `npm run capture`. Update the visual specification only if a reusable design rule changes; update the navigation handoff only if a flow or capability boundary changes. Do not copy sample statistics, answers, CEFR estimates, or simulated AI messages into the production app.
