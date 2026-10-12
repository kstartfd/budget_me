"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { cn } from "@/lib/utils"

import { useCategoryLabel } from "../category-label"
import { currentMonthKey } from "@/lib/month"

import { useSaveBudget } from "../mutations"
import {
  useBudgets,
  useCategories,
  useLimitRecommendations,
  useMonthSummary,
} from "../queries"
import { AmountDisplay } from "./amount-display"
import { BottomSheet } from "./bottom-sheet"
import { CategoryChips, CategoryChipsSkeleton } from "./category-chips"
import { Numpad, amountValueMinor } from "./numpad"
import { useBudgetUI } from "./ui-context"

/**
 * "Новый лимит / Изменить лимит" — category chips, amount + "уже потрачено",
 * the "AI советует" pill, numpad, save with the design's toast outcomes
 * (empty → info nudge, below current spend → danger, else success).
 */
export function LimitSheet() {
  const { limitSheet } = useBudgetUI()
  if (!limitSheet) return null
  // Remount per target so local state re-initializes from the sheet request.
  return (
    <LimitSheetInner
      key={`${limitSheet.categoryId}:${limitSheet.prefillMinor ?? ""}`}
      initialCategoryId={limitSheet.categoryId}
      initialAmount={
        limitSheet.prefillMinor != null ? formatBr(limitSheet.prefillMinor) : ""
      }
      editing={limitSheet.prefillMinor != null}
    />
  )
}

function LimitSheetInner({
  initialCategoryId,
  initialAmount,
  editing,
}: {
  initialCategoryId: string
  initialAmount: string
  editing: boolean
}) {
  const t = useTranslations("limits")
  const label = useCategoryLabel()
  const { closeLimit, showToast, fx } = useBudgetUI()

  const today = useToday()
  const month = currentMonthKey(today)
  const categories = useCategories()
  const summary = useMonthSummary(month)
  const budgets = useBudgets(month)
  const recs = useLimitRecommendations()
  const saveBudget = useSaveBudget()

  // Limits are for expenses only — income has no "limit" concept.
  const expenseCategories = (categories.data ?? []).filter(
    (c) => c.direction === "expense",
  )

  const [amount, setAmount] = useState(initialAmount)
  const [categoryId, setCategoryId] = useState(initialCategoryId)

  const category = categories.data?.find((c) => c.id === categoryId)
  const categoryName = category ? label(category) : ""
  const spentMinor =
    summary.data?.byCategory.find((c) => c.categoryId === categoryId)
      ?.spentMinor ?? 0
  const spentApprox = fx(spentMinor)
  const recMinor = recs.data?.[categoryId]

  // Envelope check: Σ limits should not outgrow the month's income. Show what's
  // still free to allocate; warn (but allow) when this limit pushes total over.
  const income = summary.data?.incomeMinor ?? 0
  const allocatedByOthers = (budgets.data ?? [])
    .filter((b) => b.categoryId !== categoryId)
    .reduce((sum, b) => sum + b.limitMinor, 0)
  const available = income - allocatedByOthers
  const overIncome = income > 0 && amountValueMinor(amount) > available

  function save() {
    const limitMinor = amountValueMinor(amount)
    if (limitMinor <= 0) {
      showToast("info", t("toastEnterAmount"))
      return
    }
    saveBudget.mutate(
      { categoryId, month, limitMinor },
      {
        onSuccess: () => {
          closeLimit()
          if (income > 0 && allocatedByOthers + limitMinor > income) {
            showToast(
              "danger",
              t("toastOverIncome", {
                over: formatBr(allocatedByOthers + limitMinor - income),
              }),
            )
          } else if (spentMinor > limitMinor) {
            showToast(
              "danger",
              t("toastOverAlready", {
                name: categoryName,
                spent: formatBr(spentMinor),
              }),
            )
          } else {
            showToast(
              "success",
              t("toastSet", { amount: formatBr(limitMinor), name: categoryName }),
            )
          }
        },
      },
    )
  }

  return (
    <BottomSheet
      title={editing ? t("sheetTitleEdit") : t("sheetTitleNew")}
      onClose={closeLimit}
    >
      <div className="mt-4">
        {categories.isPending ? (
          <CategoryChipsSkeleton />
        ) : (
          <CategoryChips
            categories={expenseCategories}
            selectedId={categoryId}
            onPick={setCategoryId}
          />
        )}
      </div>

      <div className="mt-4 mb-1 text-center">
        <div className="text-faint font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
          {t("limitFor")} · {categoryName}
        </div>
        <div className="mt-2">
          <AmountDisplay value={amount} size={42} />
        </div>
        <div className="text-faint mt-1 font-mono text-[12.5px] font-medium">
          {t("nowSpent")} {formatBr(spentMinor)} Br
          {spentApprox && <> · {spentApprox}</>}
        </div>
        {income > 0 && (
          <div
            className={cn(
              "mt-0.5 font-mono text-[12px] font-medium",
              overIncome ? "text-destructive" : "text-faint",
            )}
          >
            {t("availableOf", {
              available: formatBr(Math.max(available, 0)),
              income: formatBr(income),
            })}
          </div>
        )}
      </div>

      {recMinor != null && (
        <div className="my-3 text-center">
          <button
            type="button"
            onClick={() => setAmount(formatBr(recMinor))}
            className="border-[color:var(--primary-line)] bg-[color:var(--primary-wash)] text-primary inline-flex items-center gap-2 rounded-full border px-3.5 py-2 text-[12.5px] font-semibold"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2l1.9 5.6L19.5 9.5 13.9 11.4 12 17l-1.9-5.6L4.5 9.5 10.1 7.6z" />
            </svg>
            {t("aiSuggests")} {formatBr(recMinor)} Br
          </button>
        </div>
      )}

      <Numpad value={amount} onChange={setAmount} />
      <button
        type="button"
        onClick={save}
        disabled={saveBudget.isPending}
        className="bg-primary text-primary-foreground mt-3.5 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
      >
        {saveBudget.isPending ? t("saving") : t("setLimit")}
      </button>
    </BottomSheet>
  )
}

