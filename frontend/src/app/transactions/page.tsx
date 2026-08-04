"use client"

import { Suspense, useState } from "react"
import { useSearchParams } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"

import {
  TransactionsFiltersSheet,
  defaultRange,
  type TxFilters,
} from "@/features/budget/components/transactions-filters-sheet"
import { TxRow } from "@/features/budget/components/tx-row"
import {
  useCategories,
  useTransactionsRange,
  type Transaction,
} from "@/features/budget"
import { readMonthStartDay, useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { dayLabel } from "@/lib/month"
import { cn } from "@/lib/utils"
import { ListSkeleton } from "./skeleton"

export default function TransactionsPage() {
  return (
    <Suspense fallback={<ListSkeleton />}>
      <TransactionsView />
    </Suspense>
  )
}

function TransactionsView() {
  const locale = useLocale()
  const t = useTranslations("transactions")
  const tCommon = useTranslations("common")
  const searchParams = useSearchParams()

  const today = useToday()
  const startDay = readMonthStartDay()

  const [filters, setFilters] = useState<TxFilters>(() => {
    // A category link from Analytics carries the period it was looking at
    // (from/to) alongside the category — landing here on "current period"
    // instead would silently swap out what you just tapped through to see.
    const from = searchParams.get("from")
    const to = searchParams.get("to")

    return {
      ...(from && to ? { from, to } : defaultRange(today, startDay)),
      direction: "all",
      category: searchParams.get("category") ?? "all",
    }
  })
  const [filtersOpen, setFiltersOpen] = useState(false)

  const transactions = useTransactionsRange(filters.from, filters.to)
  const categories = useCategories()

  if (transactions.isPending || categories.isPending) return <ListSkeleton />
  if (transactions.isError || categories.isError) {
    return (
      <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
        {tCommon("loadError")}
      </p>
    )
  }

  // The real dictionary carries both directions, so income rows resolve their
  // category name straight from it.
  const catById = new Map(categories.data.map((c) => [c.id, c]))

  const filtered = transactions.data.filter(
    (tx) =>
      (filters.direction === "all" || tx.direction === filters.direction) &&
      (filters.category === "all" || tx.categoryId === filters.category),
  )

  const groups: { date: string; items: Transaction[] }[] = []
  for (const tx of filtered) {
    const last = groups[groups.length - 1]
    if (last && last.date === tx.date) last.items.push(tx)
    else groups.push({ date: tx.date, items: [tx] })
  }

  // Badge: how many filters differ from the defaults (range / direction / category).
  const def = defaultRange(today, startDay)
  const activeCount =
    (filters.from !== def.from || filters.to !== def.to ? 1 : 0) +
    (filters.direction !== "all" ? 1 : 0) +
    (filters.category !== "all" ? 1 : 0)

  return (
    <div className="anim-fade flex flex-col gap-4">
      <div className="flex items-baseline justify-between px-0.5">
        <h1 className="text-[22px] font-bold tracking-[-0.02em]">{t("title")}</h1>
        <span className="text-faint font-mono text-[12.5px] font-medium">
          {t("count", { count: filtered.length })}
        </span>
      </div>

      <div className="flex items-center gap-2.5 px-0.5">
        <button
          type="button"
          onClick={() => setFiltersOpen(true)}
          className="bg-card border-border text-foreground flex shrink-0 items-center gap-2 rounded-full border px-3.5 py-[7px] text-[12.5px] font-medium"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="M22 3H2l8 9.46V19l4 2v-8.54L22 3z" />
          </svg>
          {t("filters")}
          {activeCount > 0 && (
            <span className="bg-primary text-primary-foreground flex size-[18px] items-center justify-center rounded-full text-[10px] font-bold">
              {activeCount}
            </span>
          )}
        </button>
        <span className="text-faint truncate font-mono text-[11.5px]">
          {fmtDate(filters.from, locale)} – {fmtDate(filters.to, locale)}
        </span>
      </div>

      {groups.length === 0 && (
        <p className="text-muted-foreground py-12 text-center text-sm">
          {t("empty")}
        </p>
      )}

      {groups.map((group) => {
        // Split the day's total by direction so the header shows spend as «−»
        // and any income as a «+» green tag — never a «−» over a mixed sum.
        const spentMinor = group.items.reduce(
          (sum, tx) => (tx.direction === "income" ? sum : sum + tx.amountMinor),
          0,
        )
        const incomeMinor = group.items.reduce(
          (sum, tx) => (tx.direction === "income" ? sum + tx.amountMinor : sum),
          0,
        )
        return (
          <section key={group.date}>
            <div className="flex items-center justify-between px-1 pb-[9px]">
              <span className="text-faint font-mono text-[11px] font-semibold tracking-[0.05em] uppercase">
                {dayLabel(group.date, locale, tCommon("today"), tCommon("yesterday"), today)}
              </span>
              <span className="font-mono text-xs font-medium">
                {spentMinor > 0 && (
                  <span className="text-muted-foreground">
                    −{formatBr(spentMinor)} Br
                  </span>
                )}
                {incomeMinor > 0 && (
                  <span className={cn("text-success", spentMinor > 0 && "ml-2")}>
                    +{formatBr(incomeMinor)} Br
                  </span>
                )}
              </span>
            </div>
            <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[20px] border shadow-[var(--shadow-card)]">
              {group.items.map((tx) => (
                <TxRow
                  key={tx.id}
                  tx={tx}
                  category={catById.get(tx.categoryId)}
                  meta={tx.time}
                />
              ))}
            </div>
          </section>
        )
      })}

      {filtersOpen && (
        <TransactionsFiltersSheet
          categories={categories.data}
          value={filters}
          today={today}
          startDay={startDay}
          onApply={setFilters}
          onClose={() => setFiltersOpen(false)}
        />
      )}
    </div>
  )
}

function fmtDate(iso: string, locale: string): string {
  return new Date(`${iso}T00:00:00`).toLocaleDateString(locale, {
    day: "numeric",
    month: "short",
  })
}
