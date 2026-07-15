"use client"

import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"

import {
  useCategories,
  useMonthSummary,
  useTrend,
} from "@/features/budget"
import { useCategoryLabel } from "@/features/budget/category-label"
import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { currentMonthKey, daysInMonth, monthShort } from "@/lib/month"
import { AnalyticsSkeleton } from "./skeleton"

const card =
  "bg-card border-border rounded-[22px] border shadow-[var(--shadow-card)]"

export default function AnalyticsPage() {
  const locale = useLocale()
  const t = useTranslations("analytics")
  const label = useCategoryLabel()
  const tCommon = useTranslations("common")

  const today = useToday()
  const month = currentMonthKey(today)
  const summary = useMonthSummary(month)
  const categories = useCategories()
  const trend = useTrend()

  if (summary.isPending || categories.isPending || trend.isPending) {
    return <AnalyticsSkeleton />
  }
  if (summary.isError || categories.isError || trend.isError) {
    return (
      <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
        {tCommon("loadError")}
      </p>
    )
  }

  const catById = new Map(categories.data.map((c) => [c.id, c]))
  const { spentMinor, byCategory } = summary.data

  // Donut geometry per the design: r=54, stroke 17, 3px gaps, from 12 o'clock.
  const circumference = 2 * Math.PI * 54
  const segments: { color: string; dash: string; offset: string }[] = []
  let cum = 0
  for (const row of byCategory) {
    const len = (row.spentMinor / Math.max(spentMinor, 1)) * circumference
    segments.push({
      color: catById.get(row.categoryId)?.color ?? "#9198a6",
      dash: `${Math.max(len - 3, 0).toFixed(1)} ${(circumference - len + 3).toFixed(1)}`,
      offset: (-cum).toFixed(1),
    })
    cum += len
  }

  const maxTrend = Math.max(...trend.data.map((p) => p.spentMinor), 1)
  const prev = trend.data[trend.data.length - 2]
  const cur = trend.data[trend.data.length - 1]
  // Per-day pace, since the running month is partial.
  const curPace = cur.spentMinor / Number(today.slice(8, 10))
  const prevPace = prev ? prev.spentMinor / daysInMonth(prev.month) : 0
  const trendDeltaPct =
    prevPace > 0 ? ((curPace - prevPace) / prevPace) * 100 : null

  return (
    <div className="anim-fade flex flex-col gap-4">
      <h1 className="px-0.5 text-[22px] font-bold tracking-[-0.02em]">
        {t("title")}
      </h1>

      {/* donut */}
      <section className={`${card} px-5 py-[22px]`}>
        <div className="flex items-center gap-[22px]">
          <div className="relative size-[130px] shrink-0">
            <svg width="130" height="130" viewBox="0 0 140 140" className="-rotate-90">
              <circle
                cx="70"
                cy="70"
                r="54"
                fill="none"
                stroke="var(--secondary)"
                strokeWidth="17"
              />
              {segments.map((s, i) => (
                <circle
                  key={i}
                  cx="70"
                  cy="70"
                  r="54"
                  fill="none"
                  strokeWidth="17"
                  strokeLinecap="round"
                  stroke={s.color}
                  strokeDasharray={s.dash}
                  strokeDashoffset={s.offset}
                />
              ))}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-faint font-mono text-[9px] font-semibold tracking-[0.08em] uppercase">
                {t("total")}
              </span>
              <span className="mt-0.5 text-[19px] font-bold tracking-[-0.02em]">
                {formatBr(spentMinor)}
              </span>
              <span className="text-faint font-mono text-[10px]">Br</span>
            </div>
          </div>
          {/* prefetch={false} on the category rows (here and in the list below):
              every row is a Link, so Next would prefetch the transactions page
              once per category the moment the list renders — a dozen RSC
              requests on a phone connection to serve the one row that gets
              tapped. */}
          <div className="flex flex-1 flex-col gap-[9px]">
            {byCategory.map((row) => {
              const c = catById.get(row.categoryId)
              if (!c) return null
              return (
                <Link
                  key={row.categoryId}
                  href={`/transactions?category=${row.categoryId}`}
                  prefetch={false}
                  className="hover:bg-secondary/50 active:bg-secondary -mx-1 flex items-center gap-[9px] rounded-md px-1 py-0.5 transition-colors"
                >
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ background: c.color }}
                  />
                  <span className="text-muted-foreground flex-1 truncate text-[12.5px]">
                    {label(c)}
                  </span>
                  <span className="font-mono text-xs font-semibold">
                    {Math.round((row.spentMinor / Math.max(spentMinor, 1)) * 100)}%
                  </span>
                </Link>
              )
            })}
          </div>
        </div>
      </section>

      {/* trend */}
      <section className={`${card} p-5`}>
        <div className="flex items-baseline justify-between">
          <span className="text-[14.5px] font-semibold">{t("trendTitle")}</span>
          {trendDeltaPct !== null && (
            <span className="text-faint font-mono text-xs font-medium">
              {(trendDeltaPct > 0 ? "+" : "−") +
                Math.abs(trendDeltaPct).toFixed(1).replace(".", ",")}
              % {t("vsPrev")}
            </span>
          )}
        </div>
        <div className="mt-5 flex h-[118px] items-end gap-2.5">
          {trend.data.map((p, i) => {
            const current = i === trend.data.length - 1
            return (
              <div
                key={p.month}
                className="flex h-full flex-1 flex-col items-center justify-end gap-2"
              >
                <div
                  className="w-full rounded-t-[7px] rounded-b-[4px] transition-[height] duration-400"
                  style={{
                    height: `${Math.round((p.spentMinor / maxTrend) * 100)}%`,
                    background: current ? "var(--primary)" : "var(--primary-wash)",
                  }}
                />
                <span
                  className="font-mono text-[10px] font-medium"
                  style={{
                    color: current ? "var(--foreground)" : "var(--faint)",
                  }}
                >
                  {monthShort(p.month, locale)}
                </span>
              </div>
            )
          })}
        </div>
      </section>

      {/* where money goes */}
      <section>
        <div className="px-1 pb-3">
          <span className="text-[15px] font-semibold tracking-[-0.01em]">
            {t("whereMoney")}
          </span>
        </div>
        <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[20px] border shadow-[var(--shadow-card)]">
          {byCategory.map((row) => {
            const c = catById.get(row.categoryId)
            if (!c) return null
            return (
              <Link
                key={row.categoryId}
                href={`/transactions?category=${row.categoryId}`}
                prefetch={false}
                className="hover:bg-secondary/40 active:bg-secondary/60 flex items-center gap-[13px] px-4 py-3.5 transition-colors"
              >
                <span
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ background: c.color }}
                />
                <span className="flex-1 truncate text-sm font-medium">
                  {label(c)}
                </span>
                {row.deltaPct !== null && (
                  <span
                    className="min-w-11 text-right font-mono text-[11px] font-semibold"
                    style={{
                      color: row.deltaPct > 0 ? "var(--warn)" : "var(--success)",
                    }}
                  >
                    {(row.deltaPct > 0 ? "+" : "") + row.deltaPct}%
                  </span>
                )}
                <span className="w-[78px] text-right font-mono text-[13px] font-semibold">
                  {formatBr(row.spentMinor)} Br
                </span>
                <svg
                  width="15"
                  height="15"
                  viewBox="0 0 24 24"
                  fill="none"
                  aria-hidden
                  className="text-faint -mr-1 shrink-0"
                >
                  <path
                    d="M9 6l6 6-6 6"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </Link>
            )
          })}
        </div>
      </section>
    </div>
  )
}
