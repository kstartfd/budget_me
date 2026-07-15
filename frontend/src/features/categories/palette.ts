/**
 * Category colour palette — the swatches from the approved "categories mockup".
 * Each hue carries an i18n key for its swatch aria-label
 * (`categoriesPage.colors.<key>`). Hues match the design's category colours so
 * new categories sit alongside the presets without improvising colours.
 */
export const CATEGORY_PALETTE = [
  { hex: "#e0a33e", key: "amber" },
  { hex: "#35b98f", key: "green" },
  { hex: "#9b8cff", key: "violet" },
  { hex: "#5b8def", key: "blue" },
  { hex: "#e07ea8", key: "pink" },
  { hex: "#46c4c0", key: "teal" },
  { hex: "#ef6f6f", key: "red" },
  { hex: "#7c86ff", key: "periwinkle" },
  { hex: "#9198a6", key: "gray" },
] as const

/** Default hue a brand-new category starts on (first palette swatch). */
export const DEFAULT_CATEGORY_COLOR = CATEGORY_PALETTE[0].hex
