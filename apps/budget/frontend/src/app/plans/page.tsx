"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"

import { CheckIcon, ChevronLeftIcon, PlusIcon } from "@/components/app-icons"
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
import { PlansSkeleton } from "./skeleton"

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
  const plannedTotal = pending.reduce((sum, p) => sum + p.amountMinor, 0)

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
            {t("plannedTotal", { amount: formatBr(plannedTotal) })}
          </span>
        </div>
      </div>

      <button
        type="button"
        onClick={() => openPlan()}
        className="border-[color:var(--primary-line)] bg-[color:var(--primary-wash)] text-primary flex items-center justify-center gap-2 rounded-[18px] border-[1.5px] border-dashed p-[15px] text-[13.5px] font-semibold"
      >
        <PlusIcon size={16} strokeWidth={2.2} />
        {t("newPlan")}
      </button>

      {suggestions.data && suggestions.data.length > 0 && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between px-1">
            <span className="text-faint font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
              {t("suggestionsKicker")}
            </span>
            <button
              type="button"
              onClick={() => addAll(suggestions.data)}
              disabled={createPlan.isPending}
              className="text-primary text-[12px] font-semibold disabled:opacity-60"
            >
              {t("addAll")}
            </button>
          </div>

          {suggestions.data.map((suggestion) => {
            const category = catById.get(suggestion.categoryId)
            return (
              <div
                key={`${suggestion.categoryId}:${suggestion.title}`}
                className="border-border flex items-center gap-3 rounded-[16px] border border-dashed p-3"
              >
                <span
                  className="size-2.5 shrink-0 rounded-full opacity-70"
                  style={{ background: category?.color ?? "#9198a6" }}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[13px] font-semibold">
                    {suggestion.title}
                  </span>
                  <span className="text-faint block truncate text-[11.5px]">
                    {label(category)} · {formatBr(suggestion.amountMinor)} Br ·{" "}
                    {t("seenCount", { count: suggestion.seenCount })}
                  </span>
                </span>
                <button
                  type="button"
                  onClick={() => add(suggestion)}
                  disabled={createPlan.isPending}
                  aria-label={t("addSuggestion", { title: suggestion.title })}
                  className="border-[color:var(--primary-line)] text-primary flex size-7 shrink-0 items-center justify-center rounded-full border disabled:opacity-60"
                >
                  <PlusIcon size={14} strokeWidth={2.2} />
                </button>
              </div>
            )
          })}
        </div>
      )}

      {pending.length === 0 && done.length === 0 && (
        <p className="text-muted-foreground py-12 text-center text-sm">
          {t("empty")}
        </p>
      )}

      {pending.length > 0 && (
        <div className="flex flex-col gap-2.5">
          {pending.map((plan) => {
            const color = catById.get(plan.categoryId)?.color ?? "#9198a6"
            return (
              <div
                key={plan.id}
                className="bg-card border-border flex items-center gap-2 rounded-[18px] border p-[13px] shadow-[var(--shadow-card)]"
              >
                <button
                  type="button"
                  onClick={() => openPlan(plan)}
                  className="flex min-w-0 flex-1 items-center gap-3 text-left"
                >
                  <span
                    className="size-2.5 shrink-0 rounded-full"
                    style={{ background: color }}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-semibold">
                      {plan.title}
                    </span>
                    <span className="text-faint block truncate text-[11.5px]">
                      {label(catById.get(plan.categoryId))} ·{" "}
                      {formatBr(plan.amountMinor)} Br
                    </span>
                  </span>
                </button>
                <button
                  type="button"
                  onClick={() => confirm(plan)}
                  disabled={confirming}
                  className="bg-primary text-primary-foreground flex shrink-0 items-center gap-1 rounded-full px-3 py-1.5 text-[12px] font-semibold disabled:opacity-60"
                >
                  <CheckIcon size={13} />
                  {t("confirm")}
                </button>
                <button
                  type="button"
                  onClick={() => remove(plan)}
                  aria-label={t("delete")}
                  className="text-faint hover:text-destructive flex size-7 shrink-0 items-center justify-center rounded-full text-[15px] leading-none"
                >
                  ✕
                </button>
              </div>
            )
          })}
        </div>
      )}

      {done.length > 0 && (
        <>
          <div className="text-faint mt-2 px-1 font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
            {t("doneKicker")}
          </div>
          <div className="flex flex-col gap-2">
            {done.map((plan) => (
              <div
                key={plan.id}
                className="bg-card border-border flex items-center gap-3 rounded-[16px] border p-3 opacity-70 shadow-[var(--shadow-card)]"
              >
                <span className="text-success shrink-0">
                  <CheckIcon size={15} />
                </span>
                <span className="min-w-0 flex-1 truncate text-[13px] font-medium line-through">
                  {plan.title}
                </span>
                <span className="text-faint shrink-0 font-mono text-[12px]">
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
      )}
    </div>
  )
}
