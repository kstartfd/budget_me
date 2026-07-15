"use client"

import { useTranslations } from "next-intl"

import type { Category } from "./schemas"

/**
 * Resolves a category's display name: the i18n slug for the default
 * categories, the free `name` for user-added ones. One place so every screen
 * labels categories the same way.
 */
export function useCategoryLabel() {
  const t = useTranslations("categories")
  return (category: Category | undefined): string => {
    if (!category) return "—"
    return category.slug ? t(category.slug) : (category.name ?? "—")
  }
}
