# Learning sidebar design QA

Source of truth: the [canonical local design handoff](../../../docs/learning/frontend/README.md) and its [editable component](../../../docs/learning/frontend/MagicPath/LearningOverviewAssessmentCenter.tsx). The hosted MagicPath preview and the screenshot used in the earlier comparison are historical references, not acceptance authorities.

## Checked

- The live preview and local source use the same eight destinations in the same order, with the language selector above them. The application links these entries to real routes and keeps the selected language for language-specific pages.
- The sidebar is 232px wide, with 16px horizontal padding, 12px navigation labels, 15px icons, 36px navigation rows, 2px row separation and a bordered gray selected row. Logo, selector, navigation and footer use the vertical order in the MagicPath component.
- The desktop sidebar appears from the `md` breakpoint, matching the prototype. Below that width, the application uses a mobile menu and bottom navigation for the Telegram Mini App.
- The footer links to working account settings and displays the current learner's name instead of MagicPath's sample learner. Locale, theme and sign-out controls remain in Settings.

## Evidence and remaining visual difference

The local 1440 × 900 test capture is `/tmp/learning-sidebar-magicpath-alignment.png`. The sidebar structure, spacing, typography and selected state were compared with the local prototype. The app uses its installed Tabler icon set, while the prototype uses Lucide; some icon paths therefore differ. Main content also differs because the prototype contains sample content and the application reads actual learner data. The local prototype and its checked-in screenshots replace the stale hosted preview for future visual review.

## Verification

- `pnpm typecheck`: passed.
- Targeted ESLint for the shell and language selector: passed.
- Chromium navigation tests: passed, including desktop, Telegram width, language switching, starting a new plan and the 900px sidebar breakpoint.
- `git diff --check`: passed.

This QA covers the sidebar and navigation shell. It does not assert pixel identity for the full Overview content.

## P3-16 Assessment visual QA — 2026-11-27

Compared the running `learning-product-design` assessment question/result with the
real application at 1440 × 900 and 390 × 844 CSS pixels, device scale 1. Combined
captures are `/tmp/p316-compare-verified-{desktop,telegram}-{question,result}.png`;
the left side is sample prototype content, the right side is the isolated test user.
The questions and result states are comparable, but the text and data intentionally
differ: the prototype hard-codes two questions and A2, while the real adaptive test
returns its actual question and a partial, uncertain result.

- **Typography and colors:** both use compact sans-serif headings, muted copy, white
  bordered cards, a light neutral canvas and dark primary actions. The application
  uses the existing Geist stack and Tabler icons; the prototype uses Lucide.
- **Layout:** the first comparison found the application's desktop sidebar hidden
  during a question, unlike the prototype. Restored desktop navigation, mobile menu
  and the centered 720px question card, then recaptured both sizes. The app retains
  its existing mobile bottom navigation and omits the prototype-only state selector.
- **Content and actions:** the real result displays the stored grammar evidence,
  marks other skills unassessed and labels suspected gaps. It does not show the
  prototype's `Complete another section` action, which leads nowhere useful in Quick.
  A single-line input is used for the current bounded grammar answers instead of the
  prototype's sample textarea. There are no assessment content image assets.
- **Responsive and accessibility:** browser tests checked focus on the question,
  keyboard Tab to the labelled answer control, no horizontal overflow and working
  pause/resume at both widths. Captures include the full view; question controls and
  the result's skill rows were also inspected at readable scale.

No remaining blocking visual issue in this assessment slice. The 12-state live
prototype and nine-state canonical handoff still need a separate Path design sync.

final result: passed
