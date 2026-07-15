import { CategoriesScreen } from "@/features/categories"

/**
 * `/categories` — the categories management screen, reached from the Settings
 * «Категории» row. Renders inside the shared AppShell (header + nav) like every
 * other route; the screen itself is a client component.
 */
export default function CategoriesPage() {
  return <CategoriesScreen />
}
