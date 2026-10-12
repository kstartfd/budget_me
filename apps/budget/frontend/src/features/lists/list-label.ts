"use client"

import { useTranslations } from "next-intl"

import type { ItemList } from "./schemas"

/**
 * Resolves a list's display name: the i18n slug for presets (Аптечка,
 * Косметика), the free `name` for personal ones — mirrors budget's
 * useCategoryLabel exactly (same dictionary shape).
 */
export function useListLabel() {
  const t = useTranslations("lists")

  return (list: ItemList | undefined): string => {
    if (!list) return "—"

    return list.slug ? t(`preset.${list.slug}`) : (list.name ?? "—")
  }
}
