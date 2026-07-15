"use client"

import { useState } from "react"
import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"

import { CheckIcon, InfoIcon, SparkFillIcon } from "@/components/app-icons"
import { MoneyInfoSheet } from "@/features/budget/components/money-info-sheet"
import { TxRow } from "@/features/budget/components/tx-row"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { usePlanConfirm } from "@/features/budget/use-plan-confirm"
import {
  useBalance,
  useCategories,
  useInsights,
  useMonthSummary,
  usePlans,
  useTransactions,
} from "@/features/budget"
import { readMonthStartDay, useToday } from "@/features/settings"
import { formatBr, formatRate } from "@/lib/money"
import { currentMonthKey, dayLabel, daysLeftInPeriod } from "@/lib/month"
import { OverviewSkeleton } from "./skeleton"

const card =
  "bg-card border-border rounded-[24px] border shadow-[var(--shadow-card)]"

export default function OverviewPage() {
  const locale = useLocale()
  const t = useTranslations("overview")
  const tCommon = useTranslations("common")
  const tRates = useTranslations("rates")
  const tPlans = useTranslations("plans")
  const tBalance = useTranslations("balance")
  const { rate, cycleRate, fx, openBalance } = useBudgetUI()
  const { confirm, isPending: confirming } = usePlanConfirm()

  const today = useToday()
  const startDay = readMonthStartDay()
  const month = currentMonthKey(today)
  const [infoOpen, setInfoOpen] = useState(false)
  const summary = useMonthSummary(month)
  const balance = useBalance()
  const categories = useCategories()
  const transactions = useTransactions(month)
  const insights = useInsights()
  const plans = usePlans(month)

  // Transactions gate with the rest: rendering `data ?? []` on a failed query is
  // what let a broken date range show as a merely empty "recent" list.
  if (summary.isPending || categories.isPending || transactions.isPending)
    return <OverviewSkeleton />
  if (summary.isError || categories.isError || transactions.isError) {
    return (
      <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
        {tCommon("loadError")}
      </p>
    )
  }

  const { spentMinor, incomeMinor, byCategory } = summary.data
  // The real dictionary carries both directions, so income rows resolve their
  // category name straight from it; the expense-only "by category" is unaffected.
  const catById = new Map(categories.data.map((c) => [c.id, c]))
  // «Осталось» = приход − расход. Without any logged income there is no
  // baseline to spend against, so the left/over treatment is suppressed
  // (see the hint below) instead of showing a misleading "0 left / all red".
  const hasIncome = incomeMinor > 0
  const spentApprox = fx(spentMinor)
  const balanceApprox = balance.data?.set ? fx(balance.data.balanceMinor) : null
  const over = hasIncome && spentMinor > incomeMinor
  const leftMinor = Math.max(incomeMinor - spentMinor, 0)
  const daysLeft = daysLeftInPeriod(today, startDay)

  const top = byCategory.slice(0, 4)
  const maxSpent = Math.max(...top.map((c) => c.spentMinor), 1)
  // Teaser = the model's take on this month, in its own words. Absent until it
  // has been written (background job), and the card simply doesn't render.
  const advice = insights.data?.advice ?? null
  const pendingPlans = (plans.data ?? [])
    .filter((p) => !p.confirmed)
    .slice(0, 3)

  return (
    <div className="anim-fade flex flex-col gap-4 lg:grid lg:grid-cols-2 lg:gap-5">
      {/* Money on hand. Transactions alone can't know it (they never saw what was
          already in the wallet), so until the user states a figure this asks for
          one instead of showing an invented zero. Tapping it corrects the number. */}
      <button
        type="button"
        onClick={() =>
          // Only seed the numpad with a positive figure: formatBr drops the sign
          // and the numpad has no minus key, so pre-filling an overdrawn balance
          // would silently turn −150 into 150. Start empty instead.
          openBalance(
            balance.data?.set && balance.data.balanceMinor > 0
              ? formatBr(balance.data.balanceMinor)
              : null,
          )
        }
        className={`${card} flex items-center justify-between px-5 py-4 text-left lg:col-span-2 lg:px-7`}
      >
        <span className="min-w-0">
          <span className="text-faint block font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {tBalance("kicker")}
          </span>
          {balance.data?.set ? (
            <span className="mt-1 flex items-baseline gap-1.5">
              <span
                className={`text-[26px] font-bold tracking-[-0.02em] ${
                  balance.data.balanceMinor < 0 ? "text-destructive" : ""
                }`}
              >
                {balance.data.balanceMinor < 0 && "−"}
                {formatBr(balance.data.balanceMinor)}
              </span>
              <span className="text-faint text-[13px] font-semibold">Br</span>
              {balanceApprox && (
                <span className="text-muted-foreground ml-1 font-mono text-[12px]">
                  ≈ {balanceApprox}
                </span>
              )}
            </span>
          ) : (
            <span className="text-muted-foreground mt-1 block text-[13px]">
              {tBalance("notSet")}
            </span>
          )}
        </span>
        <span className="text-primary shrink-0 text-[12.5px] font-semibold">
          {balance.data?.set ? tBalance("edit") : tBalance("set")}
        </span>
      </button>

      {/* hero */}
      <section className={`${card} px-5 py-[22px] lg:col-span-2 lg:px-7 lg:py-7`}>
        <div className="flex items-center justify-between">
          <span className="flex items-center gap-1.5">
            <span className="text-faint font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
              {t("spentKicker")}
            </span>
            <button
              type="button"
              onClick={() => setInfoOpen(true)}
              aria-label={t("infoAria")}
              className="text-faint hover:text-foreground active:text-foreground -my-1 flex size-6 items-center justify-center rounded-full transition-colors"
            >
              <InfoIcon />
            </button>
          </span>
          {/* No pill when the currency is BYN — a Br-per-Br rate says nothing. */}
          {rate && (
            <button
              type="button"
              onClick={cycleRate}
              className="border-border bg-secondary flex items-center gap-1.5 rounded-full border px-2.5 py-1"
            >
              <span className="text-muted-foreground font-mono text-[11px] font-medium">
                {tRates(rate.slug)} {formatRate(rate.rate)}
              </span>
            </button>
          )}
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-[38px] font-bold tracking-[-0.03em]">
            {formatBr(spentMinor)}
          </span>
          <span className="text-faint text-base font-semibold">Br</span>
        </div>
        {(spentApprox || hasIncome) && (
          <div className="text-muted-foreground mt-0.5 font-mono text-[13px] font-medium">
            {spentApprox && <>≈ {spentApprox}</>}
            {spentApprox && hasIncome && " · "}
            {hasIncome &&
              t("fromIncome", { amount: `${formatBr(incomeMinor)} Br` })}
          </div>
        )}
        <div className="bg-secondary mt-4 h-2 overflow-hidden rounded-full">
          <div
            className="h-full rounded-full"
            style={{
              width: `${hasIncome ? Math.min((spentMinor / incomeMinor) * 100, 100) : 0}%`,
              background: over ? "var(--destructive)" : "var(--primary)",
            }}
          />
        </div>
        {hasIncome ? (
          <div className="mt-4 grid grid-cols-2 gap-2">
            {[
              [t("left"), `${formatBr(leftMinor)} Br`],
              [t("daysLeft"), String(daysLeft)],
            ].map(([label, value]) => (
              <div key={label}>
                <div className="text-faint font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
                  {label}
                </div>
                <div className="mt-[5px] text-[14.5px] font-semibold">
                  {value}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground mt-4 text-[13px] font-medium">
            {t("noIncomeHint")}
          </p>
        )}
      </section>

      {/* AI teaser */}
      {advice && (
        <Link
          href="/ai"
          className="border-[color:var(--primary-line)] flex items-start gap-3.5 rounded-[22px] border p-[18px] text-left shadow-[var(--shadow-card)] lg:col-span-2 lg:p-6"
          style={{
            background:
              "linear-gradient(150deg, var(--primary-wash), var(--card))",
          }}
        >
          <div className="bg-primary text-primary-foreground flex size-8 shrink-0 items-center justify-center rounded-[10px]">
            <SparkFillIcon />
          </div>
          <div className="min-w-0 flex-1">
            <span className="text-primary font-mono text-[10px] font-semibold tracking-[0.08em] uppercase">
              {t("aiTag")}
            </span>
            <p className="mt-[7px] line-clamp-3 text-[13.5px] leading-normal">
              {advice}
            </p>
          </div>
        </Link>
      )}

      {/* plans */}
      {pendingPlans.length > 0 && (
        <section className={`${card} p-[18px]`}>
          <div className="flex items-center justify-between pb-3">
            <span className="text-[15px] font-semibold tracking-[-0.01em]">
              {tPlans("title")}
            </span>
            <Link
              href="/plans"
              className="text-primary text-[12.5px] font-medium"
            >
              {t("seeAll")}
            </Link>
          </div>
          <div className="flex flex-col gap-3">
            {pendingPlans.map((plan) => (
              <div key={plan.id} className="flex items-center gap-3">
                <span
                  className="size-2.5 shrink-0 rounded-full"
                  style={{
                    background: catById.get(plan.categoryId)?.color ?? "#9198a6",
                  }}
                />
                <span className="min-w-0 flex-1 truncate text-sm font-medium">
                  {plan.title}
                </span>
                <span className="text-faint shrink-0 font-mono text-[13px]">
                  {formatBr(plan.amountMinor)} Br
                </span>
                <button
                  type="button"
                  onClick={() => confirm(plan)}
                  disabled={confirming}
                  aria-label={tPlans("confirm")}
                  className="bg-primary text-primary-foreground flex size-7 shrink-0 items-center justify-center rounded-full disabled:opacity-60"
                >
                  <CheckIcon size={14} />
                </button>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* by category */}
      <section className={`${card} p-[18px]`}>
        <div className="flex items-center justify-between pb-3">
          <span className="text-[15px] font-semibold tracking-[-0.01em]">
            {t("byCategory")}
          </span>
          <Link href="/analytics" className="text-primary text-[12.5px] font-medium">
            {t("seeAll")}
          </Link>
        </div>
        <div className="flex flex-col gap-3">
          {top.map((row) => {
            const c = catById.get(row.categoryId)
            if (!c) return null
            return (
              <div key={row.categoryId} className="flex items-center gap-3">
                <span
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ background: c.color }}
                />
                <span className="flex-1 truncate text-sm font-medium">
                  <CategoryName slug={c.slug} name={c.name} />
                </span>
                <span className="bg-secondary h-1.5 w-16 overflow-hidden rounded-full">
                  <span
                    className="block h-full rounded-full"
                    style={{
                      width: `${Math.round((row.spentMinor / maxSpent) * 100)}%`,
                      background: c.color,
                    }}
                  />
                </span>
                <span className="w-[82px] text-right font-mono text-[13px] font-medium">
                  {formatBr(row.spentMinor)} Br
                </span>
              </div>
            )
          })}
        </div>
      </section>

      {/* recent */}
      <section className={`${card} overflow-hidden`}>
        <div className="flex items-center justify-between px-4 pt-4 pb-1">
          <span className="text-[15px] font-semibold tracking-[-0.01em]">
            {t("recent")}
          </span>
          <Link
            href="/transactions"
            className="text-primary text-[12.5px] font-medium"
          >
            {t("seeAll")}
          </Link>
        </div>
        <div className="divide-rowline divide-y">
          {transactions.data.slice(0, 3).map((tx) => (
            <TxRow
              key={tx.id}
              tx={tx}
              category={catById.get(tx.categoryId)}
              meta={dayLabel(tx.date, locale, tCommon("today"), tCommon("yesterday"), today)}
            />
          ))}
        </div>
      </section>

      {infoOpen && <MoneyInfoSheet onClose={() => setInfoOpen(false)} />}
    </div>
  )
}

function CategoryName({ slug, name }: { slug?: string; name?: string }) {
  const t = useTranslations("categories")
  return <>{slug ? t(slug) : (name ?? "—")}</>
}
