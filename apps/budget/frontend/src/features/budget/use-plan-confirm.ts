"use client"

import { useTranslations } from "next-intl"

import { useToday } from "@/features/settings"
import { nowTime } from "@/lib/month"

import { useBudgetUI } from "./components/ui-context"
import { useAddTransaction, useConfirmPlan } from "./mutations"
import type { Plan } from "./schemas"

/**
 * Confirm a plan: create the real (expense) transaction from it, then stamp the
 * plan confirmed. Two chained writes — good enough at this scale; the rare case
 * of the confirm failing after the transaction lands just leaves the plan
 * pending (re-confirm would duplicate, so users delete it instead).
 */
export function usePlanConfirm() {
  const t = useTranslations("plans")
  const today = useToday()
  const addTransaction = useAddTransaction()
  const confirmPlan = useConfirmPlan()
  const { showToast } = useBudgetUI()

  const isPending = addTransaction.isPending || confirmPlan.isPending

  function confirm(plan: Plan) {
    addTransaction.mutate(
      {
        amountMinor: plan.amountMinor,
        categoryId: plan.categoryId,
        title: plan.title,
        date: today,
        time: nowTime(),
        direction: "expense",
        counted: true,
      },
      {
        onSuccess: () => {
          confirmPlan.mutate(plan.id, {
            onSuccess: () =>
              showToast("success", t("confirmedToast", { title: plan.title })),
          })
        },
      },
    )
  }

  return { confirm, isPending }
}
