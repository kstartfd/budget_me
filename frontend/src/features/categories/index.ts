/**
 * Public surface of the categories feature — the management screen and its
 * palette. Category data + write hooks live in the budget feature (it owns the
 * category cache); this feature is the settings sub-screen that composes them.
 */

export { CategoriesScreen } from "./components/categories-screen"
export { CATEGORY_PALETTE, DEFAULT_CATEGORY_COLOR } from "./palette"
export type { CategoryFormValues } from "./schemas"
