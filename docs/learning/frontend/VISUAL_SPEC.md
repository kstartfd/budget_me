# Learning visual specification

This specification records reusable rules from the [editable prototype source](MagicPath/LearningOverviewAssessmentCenter.tsx). Use that source and the [local preview](README.md#run-without-magicpath) for exact component anatomy; the [screen captures](screenshots/README.md) show representative states. All prototype data is illustrative.

## Layout and hierarchy

- Light workspace canvas `#f8f8f7`, foreground `#282828`. White `#fff` cards have a subtle `#e6e6e3` border and approximately 12px radius. Muted supporting text uses `#777773`.
- Desktop sidebar: 232px wide, full viewport height, white, right divider `#e7e7e5`; 16px horizontal and 20px vertical padding. The logo sits above the language selector, then the eight navigation items in the order in [Navigation Handoff](NAVIGATION_HANDOFF.md). The learner/account area stays at the bottom.
- Main content: fluid, centered, up to 1490px, with 16px horizontal padding on the smallest viewport, 24px from `sm`, and 24px top padding from `md`. Overview uses a two-column card composition where space allows; content becomes a single column on narrow screens.
- Page hierarchy: 11px breadcrumb/caption, approximately 23px page title, 13px explanatory line, then content cards. Use short actionable labels and quiet empty states rather than sample progress as a default.

## Navigation and components

- Navigation rows are 36px high with 2px separation, 12px medium labels, 15px outline icons, 10px icon/text gap, and 12px horizontal row padding. The current row has `#f0f0ee` fill and `#cececb` border. Hover uses a very light neutral fill. The source uses Lucide icons.
- The language selector is a bordered white card above the navigation; its code badge, language name, helper line, and dropdown chevron form one control. Changing it changes the selected language for the workspace.
- Content cards use restrained borders and spacing. Primary actions are dark `#292929` buttons with white text; secondary actions are outlined. Status chips distinguish learning blue (`#5069ee`), mastery green (`#299141`), due/overdue amber, and unknown neutral. These colors supplement explicit text.
- The Knowledge Base uses domain tabs, a category tree, a search field, status chips, sort/filter controls, and rows with Preview/Practice actions. The separate personal notebook must remain labelled separately in the application.

## Responsive behavior

- At **768px and wider**, show the persistent sidebar. Below 768px, show the menu button and a 232px slide-over navigation panel with a backdrop. The design source does not include the production app's bottom tab bar; the app may retain that additional Telegram navigation aid if the eight destinations remain discoverable.
- Review at **1440 × 900** for desktop and **390 × 844** for Telegram-width mobile; those are the checked-in screenshot viewports. The mobile layout stacks cards, keeps the primary action reachable, and avoids horizontal scrolling. Long pages such as Knowledge Base scroll vertically.

## Typography and accessibility

- Font stack: Geist, Inter, then system sans-serif. The prototype uses mostly 11–14px supporting text and 20–23px major headings. Preserve visual contrast and readable text at mobile widths when implementing in the app.
- Navigation and actions use semantic buttons/links, visible focus indication, one `aria-current="page"` destination, labelled selects/fields, and text labels for state. Keep keyboard order consistent with visual order. Opening the mobile navigation should move focus inside and closing should return focus to the trigger; the plain prototype does **not** yet implement a complete focus trap or Escape handling, so the application must supply that behavior.
- Do not convey skill evidence, due status, or assessment availability through color alone. Unknown and insufficient-data states need text and must not display a fabricated level.
