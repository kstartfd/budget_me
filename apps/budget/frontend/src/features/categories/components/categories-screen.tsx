"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { ChevronLeftIcon, PlusIcon } from "@/components/app-icons"
import { Skeleton } from "@/components/ui/skeleton"
import { useCategoryLabel } from "@/features/budget/category-label"
import {
  useBudgetUI,
  useCategories,
  useDeleteCategory,
  type Category,
  type Direction,
} from "@/features/budget"
import { cn } from "@/lib/utils"

import { CategoryRow } from "./category-row"
import {
  CategorySheet,
  type CategorySheetRequest,
} from "./category-sheet"

/**
 * Categories management screen (design PHONE 4 · SETTINGS "categories" +
 * WEB FRAME 2 "categories table"): a Расходы/Доходы segment over the two
 * category lists. Both come from the real backend (GET /categories serves both
 * directions), filtered here on `direction`; presets are locked, personal rows
 * edit and delete. Add/edit runs through CategorySheet (bottom sheet on mobile,
 * dialog on desktop). Reached from the Settings «Категории» row.
 */
export function CategoriesScreen() {
  const t = useTranslations("categoriesPage")
  const tCommon = useTranslations("common")
  const label = useCategoryLabel()
  const { showToast } = useBudgetUI()

  const [kind, setKind] = useState<Direction>("expense")
  const [sheet, setSheet] = useState<CategorySheetRequest | null>(null)

  const categories = useCategories()
  const remove = useDeleteCategory()

  // One dictionary, both directions — filter it to the active tab.
  const list = (categories.data ?? []).filter((c) => c.direction === kind)
  // Presets/shared categories carry an i18n slug and no personal owner; those
  // are locked. Personal categories (free `name`, no slug) edit and delete.
  const isPreset = (c: Category) => c.slug != null

  function handleRowDelete(category: Category) {
    remove.mutate(
      { id: category.id, kind },
      {
        onSuccess: () =>
          showToast("info", t("deleted", { name: label(category) })),
      },
    )
  }

  return (
    <div className="anim-fade flex flex-col gap-[18px]">
      {/* breadcrumb + title */}
      <div className="flex items-start gap-2.5">
        <Link
          href="/settings"
          aria-label={t("crumb")}
          className="border-border bg-secondary text-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px] border"
        >
          <ChevronLeftIcon />
        </Link>
        <div className="min-w-0">
          <div className="text-faint truncate font-mono text-[11px]">
            {t("crumb")} › {t("title")}
          </div>
          <h1 className="text-[22px] font-bold tracking-[-0.02em]">
            {t("title")}
          </h1>
          <p className="text-muted-foreground mt-1 hidden text-[13px] lg:block">
            {t("desc")}
          </p>
        </div>
      </div>

      {/* Расходы / Доходы segment */}
      <div
        role="tablist"
        aria-label={t("title")}
        className="bg-secondary flex gap-1 rounded-2xl p-1 lg:self-start"
      >
        {(["expense", "income"] as const).map((k) => {
          const on = kind === k
          return (
            <button
              key={k}
              type="button"
              role="tab"
              aria-selected={on}
              onClick={() => setKind(k)}
              className={cn(
                "flex-1 rounded-[13px] py-[9px] text-[13.5px] font-semibold transition-colors lg:min-w-[120px] lg:flex-none",
                on
                  ? "bg-card text-foreground shadow-[var(--shadow-card)]"
                  : "text-faint",
              )}
            >
              {t(k === "expense" ? "expenseTab" : "incomeTab")}
            </button>
          )
        })}
      </div>

      {/* section label + count */}
      <div className="-mb-1.5 flex items-baseline justify-between px-0.5">
        <span className="text-faint text-[11px] font-bold tracking-[0.06em] uppercase">
          {t(kind === "expense" ? "expenseSection" : "incomeSection")}
        </span>
        <span className="text-faint font-mono text-[11px]">{list.length}</span>
      </div>

      {/* list */}
      {categories.isPending ? (
        <ListSkeleton />
      ) : categories.isError ? (
        <p
          role="alert"
          className="text-muted-foreground py-10 text-center text-sm"
        >
          {tCommon("loadError")}
        </p>
      ) : list.length === 0 ? (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {t("empty")}
        </p>
      ) : (
        <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[22px] border shadow-[var(--shadow-card)]">
          {list.map((c) => (
            <CategoryRow
              key={c.id}
              category={c}
              name={label(c)}
              isPreset={isPreset(c)}
              onEdit={() => setSheet({ kind, category: c })}
              onDelete={() => handleRowDelete(c)}
            />
          ))}
        </div>
      )}

      {/* add */}
      <button
        type="button"
        onClick={() => setSheet({ kind })}
        className="bg-primary text-primary-foreground flex w-full items-center justify-center gap-2 rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] lg:max-w-[260px]"
      >
        <PlusIcon />
        {t("add")}
      </button>

      {sheet && (
        <CategorySheet
          request={sheet}
          categoryName={sheet.category ? label(sheet.category) : ""}
          onClose={() => setSheet(null)}
        />
      )}
    </div>
  )
}

function ListSkeleton() {
  return (
    <div
      aria-hidden
      className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[22px] border"
    >
      {Array.from({ length: 5 }, (_, i) => (
        <div key={i} className="flex items-center gap-[13px] px-[15px] py-[13px]">
          <Skeleton className="size-[30px] shrink-0 rounded-full" />
          <Skeleton className="h-4 w-28 rounded" />
        </div>
      ))}
    </div>
  )
}
