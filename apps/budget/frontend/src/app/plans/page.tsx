"use client"

import { useCallback, useState } from "react"
import { useTranslations } from "next-intl"

import { useProvideAddAction } from "@/components/add-action"
import { CheckIcon, ChevronDownIcon, PlusIcon } from "@/components/app-icons"
import { useCategoryLabel } from "@/features/budget/category-label"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { usePlanConfirm } from "@/features/budget/use-plan-confirm"
import {
  useCategories,
  useCreatePlan,
  useDeletePlan,
  usePlans,
  usePlanSuggestions,
  type Plan,
  type PlanSuggestion,
} from "@/features/budget"
import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { currentMonthKey } from "@/lib/month"
import { cn } from "@/lib/utils"
import { PlansSkeleton } from "./skeleton"

type Tab = "active" | "drafts" | "done"

const TABLE_COLS = "lg:grid-cols-[1fr_160px_150px_130px]"

/**
 * Plans (design «budget_me Планы», 1a desktop / 1b Telegram): one set of
 * sections, laid out two ways. Desktop shows them stacked — drafts, the table
 * of active plans, a collapsible «Выполнено» footer. On a phone a segmented
 * control shows one section at a time; the `lg:` classes override it, so no
 * JS breakpoint is needed.
 */
export default function PlansPage() {
  const t = useTranslations("plans")
  const tCommon = useTranslations("common")
  const label = useCategoryLabel()
  const { openPlan, showToast } = useBudgetUI()

  const today = useToday()
  const month = currentMonthKey(today)
  const plans = usePlans(month)
  const categories = useCategories()
  const suggestions = usePlanSuggestions(month)
  const createPlan = useCreatePlan()
  const deletePlan = useDeletePlan()
  const { confirm, isPending: confirming } = usePlanConfirm()
  const [tab, setTab] = useState<Tab>("active")
  const [doneOpen, setDoneOpen] = useState(false)

  useProvideAddAction(useCallback(() => openPlan(), [openPlan]))

  if (plans.isPending || categories.isPending) return <PlansSkeleton />
  if (plans.isError || categories.isError) {
    return (
      <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
        {tCommon("loadError")}
      </p>
    )
  }

  const catById = new Map(categories.data.map((c) => [c.id, c]))
  const pending = plans.data.filter((p) => !p.confirmed)
  const done = plans.data.filter((p) => p.confirmed)
  const drafts = suggestions.data ?? []
  const plannedTotal = pending.reduce((sum, p) => sum + p.amountMinor, 0)
  const doneTotal = done.reduce((sum, p) => sum + p.amountMinor, 0)

  const tabs: { id: Tab; count: number }[] = [
    { id: "active", count: pending.length },
    { id: "drafts", count: drafts.length },
    { id: "done", count: done.length },
  ]

  function remove(plan: Plan) {
    deletePlan.mutate(plan.id, {
      onSuccess: () => showToast("info", t("removed", { title: plan.title })),
    })
  }

  function planFrom(suggestion: PlanSuggestion) {
    return {
      month,
      categoryId: suggestion.categoryId,
      title: suggestion.title,
      amountMinor: suggestion.amountMinor,
    }
  }

  function add(suggestion: PlanSuggestion) {
    createPlan.mutate(planFrom(suggestion), {
      onSuccess: () =>
        showToast("success", t("suggestionAdded", { title: suggestion.title })),
    })
  }

  function addAll(all: PlanSuggestion[]) {
    all.forEach((suggestion) => createPlan.mutate(planFrom(suggestion)))
    showToast("success", t("suggestionsAdded", { count: all.length }))
  }

  return (
    <div className="anim-fade flex flex-col gap-4 lg:gap-[26px]">
      <header className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-[26px] font-bold tracking-[-0.02em] lg:text-[28px]">
            {t("title")}
          </h1>
          <p className="mt-1.5 flex items-baseline gap-2 lg:hidden">
            <span className="font-mono text-[20px]">
              {formatBr(plannedTotal)} Br
            </span>
            <span className="text-faint text-[13px]">{t("plannedCaption")}</span>
          </p>
          <p className="text-muted-foreground mt-1 hidden text-sm lg:block">
            {t("planned")}{" "}
            <span className="text-foreground font-mono">
              {formatBr(plannedTotal)} Br
            </span>{" "}
            · {t("activeCount", { count: pending.length })}
          </p>
        </div>

        <button
          type="button"
          onClick={() => openPlan()}
          className="bg-primary text-primary-foreground hidden h-11 shrink-0 items-center gap-2 rounded-xl px-5 text-sm font-semibold lg:flex"
        >
          <PlusIcon size={16} strokeWidth={2.2} />
          {t("newPlan")}
        </button>
      </header>

      <div
        role="tablist"
        aria-label={t("title")}
        className="bg-secondary flex gap-1 rounded-xl p-1 lg:hidden"
      >
        {tabs.map(({ id, count }) => {
          const on = id === tab
          return (
            <button
              key={id}
              type="button"
              role="tab"
              aria-selected={on}
              onClick={() => setTab(id)}
              className={cn(
                "flex-1 rounded-[9px] py-2 text-[13px] font-medium transition-colors",
                on
                  ? "text-primary bg-[color:var(--primary-wash)] ring-1 ring-[color:var(--primary-line)] ring-inset"
                  : "text-faint",
              )}
            >
              {t(`tabs.${id}`)} {count}
            </button>
          )
        })}
      </div>

      <section
        className={cn(
          "flex-col gap-2.5",
          tab === "drafts" ? "flex" : "hidden",
          drafts.length > 0 ? "lg:flex" : "lg:hidden",
        )}
      >
        {drafts.length > 0 ? (
          <>
            <div className="flex items-center justify-between px-1">
              <span className="text-faint hidden font-mono text-[11px] font-semibold tracking-[0.1em] uppercase lg:inline">
                {t("suggestionsKicker")}
              </span>
              <button
                type="button"
                onClick={() => addAll(drafts)}
                disabled={createPlan.isPending}
                className="text-primary ml-auto text-[13px] font-semibold disabled:opacity-60"
              >
                {t("addAll")}
              </button>
            </div>

            <div className="grid gap-2 lg:grid-cols-3 lg:gap-3">
              {drafts.map((suggestion) => {
                const category = catById.get(suggestion.categoryId)
                return (
                  <div
                    key={`${suggestion.categoryId}:${suggestion.title}`}
                    className="border-border flex items-center justify-between gap-2.5 rounded-[14px] border border-dashed px-4 py-3.5"
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold">
                        {suggestion.title}
                      </span>
                      <span className="text-faint mt-[3px] block truncate text-xs">
                        {label(category)} · {formatBr(suggestion.amountMinor)} Br ·{" "}
                        {t("seenCount", { count: suggestion.seenCount })}
                      </span>
                    </span>
                    <button
                      type="button"
                      onClick={() => add(suggestion)}
                      disabled={createPlan.isPending}
                      aria-label={t("addSuggestion", { title: suggestion.title })}
                      className="border-[color:var(--primary-line)] text-primary flex size-[30px] shrink-0 items-center justify-center rounded-full border disabled:opacity-60"
                    >
                      <PlusIcon size={14} strokeWidth={2.2} />
                    </button>
                  </div>
                )
              })}
            </div>
          </>
        ) : (
          <p className="text-muted-foreground py-12 text-center text-sm">
            {t("noDrafts")}
          </p>
        )}
      </section>

      <section
        className={cn(
          "flex-col gap-2 lg:flex lg:gap-0 lg:overflow-hidden lg:rounded-2xl lg:border lg:border-border lg:bg-card",
          tab === "active" ? "flex" : "hidden",
        )}
      >
        <div
          className={cn(
            "text-faint border-border hidden border-b px-5 py-3 font-mono text-[11px] tracking-[0.1em] uppercase lg:grid",
            TABLE_COLS,
          )}
        >
          <span>{t("tabs.active")}</span>
          <span>{t("category")}</span>
          <span className="text-right">{t("colAmount")}</span>
          <span />
        </div>

        {pending.length === 0 && (
          <p className="text-muted-foreground py-12 text-center text-sm">
            {t("empty")}
          </p>
        )}

        {pending.map((plan) => {
          const category = catById.get(plan.categoryId)
          return (
            <div
              key={plan.id}
              className={cn(
                "bg-card border-border flex items-center gap-3 rounded-[14px] border py-3 pr-3 pl-3.5",
                "lg:hover:bg-secondary/40 lg:grid lg:gap-0 lg:rounded-none lg:border-0 lg:border-b lg:px-5 lg:py-[13px] lg:last:border-b-0",
                TABLE_COLS,
              )}
            >
              <button
                type="button"
                onClick={() => openPlan(plan)}
                className="flex min-w-0 flex-1 items-center gap-3 text-left"
              >
                <span
                  className="size-2 shrink-0 rounded-full"
                  style={{ background: category?.color ?? "#9198a6" }}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold">
                    {plan.title}
                  </span>
                  <span className="text-faint mt-0.5 block truncate text-xs lg:hidden">
                    {label(category)} ·{" "}
                    <span className="font-mono">{formatBr(plan.amountMinor)} Br</span>
                  </span>
                </span>
              </button>

              <span className="text-muted-foreground hidden truncate text-[13px] lg:block">
                {label(category)}
              </span>
              <span className="hidden text-right font-mono text-[13px] lg:block">
                {formatBr(plan.amountMinor)} Br
              </span>

              <div className="flex shrink-0 items-center justify-end gap-2 lg:gap-3">
                <button
                  type="button"
                  onClick={() => confirm(plan)}
                  disabled={confirming}
                  aria-label={t("confirm")}
                  className="bg-primary text-primary-foreground flex size-10 items-center justify-center rounded-full text-[13px] font-semibold disabled:opacity-60 lg:size-auto lg:px-3.5 lg:py-1.5"
                >
                  <span className="lg:hidden">
                    <CheckIcon size={16} />
                  </span>
                  <span className="hidden lg:inline">{t("confirm")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => remove(plan)}
                  aria-label={t("delete")}
                  className="text-faint hover:text-destructive flex size-7 items-center justify-center rounded-full text-[15px] leading-none"
                >
                  ✕
                </button>
              </div>
            </div>
          )
        })}
      </section>

      <section
        className={cn(
          "flex-col gap-2.5",
          tab === "done" ? "flex" : "hidden",
          done.length > 0 ? "lg:flex" : "lg:hidden",
        )}
      >
        {done.length > 0 ? (
          <>
            <button
              type="button"
              onClick={() => setDoneOpen((open) => !open)}
              aria-expanded={doneOpen}
              className="text-faint hidden items-center justify-between px-1 font-mono text-xs tracking-[0.1em] uppercase lg:flex"
            >
              <span>{t("doneHeading", { count: done.length })}</span>
              <span className="flex items-center gap-1.5 tracking-normal normal-case">
                {formatBr(doneTotal)} Br
                <span
                  className={cn("transition-transform", doneOpen && "rotate-180")}
                >
                  <ChevronDownIcon size={14} />
                </span>
              </span>
            </button>

            <div
              className={cn(
                "flex flex-col gap-2",
                doneOpen ? "lg:flex" : "lg:hidden",
              )}
            >
              {done.map((plan) => (
                <div
                  key={plan.id}
                  className="bg-card border-border flex items-center gap-3 rounded-[14px] border p-3 opacity-70 shadow-[var(--shadow-card)]"
                >
                  <span className="text-success shrink-0">
                    <CheckIcon size={15} />
                  </span>
                  <span className="min-w-0 flex-1 truncate text-[13px] font-medium line-through">
                    {plan.title}
                  </span>
                  <span className="text-faint shrink-0 font-mono text-xs">
                    {formatBr(plan.amountMinor)} Br
                  </span>
                  <button
                    type="button"
                    onClick={() => remove(plan)}
                    aria-label={t("delete")}
                    className="text-faint hover:text-destructive flex size-6 shrink-0 items-center justify-center rounded-full text-[13px] leading-none"
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          </>
        ) : (
          <p className="text-muted-foreground py-12 text-center text-sm">
            {t("noDone")}
          </p>
        )}
      </section>
    </div>
  )
}
