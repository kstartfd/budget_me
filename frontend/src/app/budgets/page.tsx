"use client"

import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"

import {
  ChevronLeftIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "@/components/app-icons"
import { useCategoryLabel } from "@/features/budget/category-label"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { wash } from "@/features/budget/components/wash"
import {
  useBudgets,
  useCategories,
  useDeleteBudget,
  useMonthSummary,
} from "@/features/budget"
import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { currentMonthKey, monthShort } from "@/lib/month"
import { LimitsSkeleton } from "./skeleton"

const statusColor = {
  ok: "#35b98f",
  near: "#e0a33e",
  over: "#ea6a6a",
} as const

export default function LimitsPage() {
  const t = useTranslations("limits")
  const locale = useLocale()
  const label = useCategoryLabel()
  const tCommon = useTranslations("common")
  const { openLimit, showToast } = useBudgetUI()

  const today = useToday()
  const month = currentMonthKey(today)
  const categories = useCategories()
  const budgets = useBudgets(month)
  const summary = useMonthSummary(month)
  const deleteBudget = useDeleteBudget()

  if (categories.isPending || budgets.isPending || summary.isPending) {
    return <LimitsSkeleton />
  }
  if (categories.isError || budgets.isError || summary.isError) {
    return (
      <p
        role="alert"
        className="text-muted-foreground py-10 text-center text-sm"
      >
        {tCommon("loadError")}
      </p>
    )
  }

  // Limits apply to expenses only — income has no "limit" concept.
  const expenseCategories = categories.data.filter(
    (c) => c.direction === "expense",
  )

  const limitByCategory = new Map(
    budgets.data.map((b) => [b.categoryId, b.limitMinor]),
  )
  // The month each limit started applying from. Anything other than the month on
  // screen means it was inherited rather than set here — worth saying out loud,
  // since the whole point is that limits no longer need re-entering.
  const validFromByCategory = new Map(
    budgets.data.map((b) => [b.categoryId, b.validFrom]),
  )
  const spentByCategory = new Map(
    summary.data.byCategory.map((c) => [c.categoryId, c.spentMinor]),
  )

  const overCount = expenseCategories.filter((c) => {
    const lim = limitByCategory.get(c.id)
    return lim != null && (spentByCategory.get(c.id) ?? 0) > lim
  }).length
  const unsetCount = expenseCategories.filter(
    (c) => !limitByCategory.has(c.id),
  ).length
  const summaryLabel =
    overCount > 0
      ? t("summaryOver", { count: overCount })
      : unsetCount > 0
        ? t("summaryUnset", { count: unsetCount })
        : t("summaryOk")

  function removeLimit(categoryId: string, name: string) {
    deleteBudget.mutate(
      { categoryId, month },
      { onSuccess: () => showToast("info", t("toastRemoved", { name })) },
    )
  }

  return (
    <div className="anim-fade flex flex-col gap-4">
      <div className="flex items-start gap-2.5">
        <Link
          href="/settings"
          aria-label={t("crumb")}
          className="border-border bg-secondary text-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px] border"
        >
          <ChevronLeftIcon />
        </Link>
        <div className="flex min-w-0 grow items-end justify-between gap-2">
          <div className="min-w-0">
            <div className="text-faint truncate font-mono text-[11px]">
              {t("crumb")} › {t("title")}
            </div>
            <h1 className="text-[22px] font-bold tracking-[-0.02em]">
              {t("title")}
            </h1>
          </div>
          <span className="text-faint shrink-0 font-mono text-[12.5px] font-medium">
            {summaryLabel}
          </span>
        </div>
      </div>

      <button
        type="button"
        onClick={() => openLimit(expenseCategories[0]?.id ?? "")}
        className="border-[color:var(--primary-line)] bg-[color:var(--primary-wash)] text-primary flex items-center justify-center gap-2 rounded-[18px] border-[1.5px] border-dashed p-[15px] text-[13.5px] font-semibold"
      >
        <PlusIcon size={16} strokeWidth={2.2} />
        {t("newLimit")}
      </button>

      <p className="text-faint -mt-1 px-0.5 text-[11.5px] font-medium">
        {t("carriedHint")}
      </p>

      <div className="flex flex-col gap-3">
        {expenseCategories.map((category) => {
          const name = label(category)
          const spent = spentByCategory.get(category.id) ?? 0
          const limit = limitByCategory.get(category.id)

          if (limit == null) {
            return (
              <div
                key={category.id}
                className="bg-card border-border rounded-[20px] border px-[18px] py-[17px] shadow-[var(--shadow-card)]"
              >
                <CardHead name={name} color={category.color}>
                  <span className="bg-secondary text-faint rounded-full px-2.5 py-[3px] font-mono text-[11px] font-semibold">
                    {t("pillNoLimit")}
                  </span>
                </CardHead>
                <div className="mt-[11px] flex items-center justify-between gap-2.5">
                  <span className="text-faint font-mono text-[12.5px] font-medium">
                    {formatBr(spent)} Br
                  </span>
                  <button
                    type="button"
                    onClick={() => openLimit(category.id)}
                    className="border-[color:var(--primary-line)] bg-[color:var(--primary-wash)] text-primary rounded-xl border px-3.5 py-2 text-[12.5px] font-semibold whitespace-nowrap"
                  >
                    {t("setLimitShort")}
                  </button>
                </div>
              </div>
            )
          }

          const pct = spent / limit
          const status = pct > 1 ? "over" : pct > 0.85 ? "near" : "ok"
          const color = statusColor[status]
          const remaining = limit - spent

          return (
            <div
              key={category.id}
              className="bg-card border-border rounded-[20px] border px-[18px] py-[17px] shadow-[var(--shadow-card)]"
            >
              <CardHead name={name} color={category.color}>
                <span
                  className="rounded-full px-2.5 py-[3px] font-mono text-[11px] font-semibold"
                  style={{ background: wash(color), color }}
                >
                  {status === "over"
                    ? t("pillOver")
                    : status === "near"
                      ? t("pillNear")
                      : `${Math.round(pct * 100)}%`}
                </span>
              </CardHead>
              <div className="bg-secondary mt-[13px] h-[9px] overflow-hidden rounded-full">
                <div
                  className="h-full rounded-full"
                  style={{
                    width: `${Math.min(pct * 100, 100)}%`,
                    background: color,
                  }}
                />
              </div>
              <div className="mt-[9px] flex items-baseline justify-between">
                <span className="text-muted-foreground font-mono text-[12.5px] font-medium">
                  {formatBr(spent)} {t("of")} {formatBr(limit)} Br
                </span>
                <span
                  className="font-mono text-xs font-medium"
                  style={{
                    color:
                      status === "over"
                        ? "var(--destructive)"
                        : "var(--muted-foreground)",
                  }}
                >
                  {status === "over" ? "−" : ""}
                  {formatBr(Math.abs(remaining))} Br
                </span>
              </div>
              {(() => {
                const from = validFromByCategory.get(category.id)

                return from && from !== month ? (
                  <div className="text-faint mt-[7px] font-mono text-[11px] font-medium">
                    {t("carriedSince", { month: monthShort(from, locale) })}
                  </div>
                ) : null
              })()}
              <div className="border-rowline mt-[13px] flex gap-[9px] border-t pt-3">
                <button
                  type="button"
                  onClick={() => openLimit(category.id, limit)}
                  className="border-border bg-secondary flex flex-1 items-center justify-center gap-1.5 rounded-xl border p-[9px] text-[12.5px] font-semibold"
                >
                  <PencilIcon />
                  {t("editShort")}
                </button>
                <button
                  type="button"
                  onClick={() => removeLimit(category.id, name)}
                  disabled={deleteBudget.isPending}
                  className="border-border text-destructive flex items-center justify-center gap-1.5 rounded-xl border px-[15px] py-[9px] text-[12.5px] font-semibold disabled:opacity-60"
                >
                  <TrashIcon />
                  {t("deleteShort")}
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function CardHead({
  name,
  color,
  children,
}: {
  name: string
  color: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center gap-[11px]">
      <span
        className="size-[11px] shrink-0 rounded-full"
        style={{ background: color }}
      />
      <span className="flex-1 truncate text-[14.5px] font-semibold">
        {name}
      </span>
      {children}
    </div>
  )
}
