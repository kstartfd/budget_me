"use client"

import { useState, type CSSProperties } from "react"
import { useTranslations } from "next-intl"

import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { currentMonthKey } from "@/lib/month"

import { useCategoryLabel } from "../category-label"
import { useCreatePlan, useUpdatePlan } from "../mutations"
import { useCategories } from "../queries"
import type { Plan } from "../schemas"
import { AmountDisplay } from "./amount-display"
import { BottomSheet } from "./bottom-sheet"
import { CategoryChips, CategoryChipsSkeleton } from "./category-chips"
import { Numpad, amountValueMinor } from "./numpad"
import { useBudgetUI } from "./ui-context"

/**
 * "Новый план / Изменить план" — a planned expense: an expense category, an
 * amount (numpad) and a title. Opened from the Plans screen and the dashboard
 * card. Remounted per target so state re-initialises from the sheet request.
 *
 * Two steps, mirroring the add sheet (3b · TWO STEP), for one reason: the title
 * used to sit in the same sheet as the numpad, so focusing it opened the OS
 * keyboard under an already tall sheet and threw the whole panel up the screen.
 *
 * Step 1 (amount): category chips, the custom numpad, «Сохранить». No text input
 * lives here, so the OS keyboard never appears. Step 2 (title): the free-text
 * title — the only place the keyboard shows — over a short summary, so there is
 * room for it. The title is optional here and falls back to the category name
 * (the backend rejects an empty one), exactly like the add sheet's note.
 */
export function PlanSheet() {
  const { planSheet } = useBudgetUI()

  if (!planSheet) return null

  return <PlanSheetInner key={planSheet.plan?.id ?? "new"} plan={planSheet.plan} />
}

function PlanSheetInner({ plan }: { plan: Plan | null }) {
  const t = useTranslations("plans")
  const label = useCategoryLabel()
  const { closePlan, showToast } = useBudgetUI()

  const today = useToday()
  const month = currentMonthKey(today)
  const categories = useCategories()
  const createPlan = useCreatePlan()
  const updatePlan = useUpdatePlan()

  const editing = plan !== null
  const [title, setTitle] = useState(plan?.title ?? "")
  const [amount, setAmount] = useState(plan ? formatBr(plan.amountMinor) : "")
  const [categoryId, setCategoryId] = useState(plan?.categoryId ?? "")
  const [step, setStep] = useState<1 | 2>(1)
  const [nav, setNav] = useState<"fwd" | "back">("fwd")

  // Plans are expenses only; a stale pick (or none yet) falls back to the first.
  const expenseCategories = (categories.data ?? []).filter(
    (c) => c.direction === "expense",
  )
  const selectedId = expenseCategories.some((c) => c.id === categoryId)
    ? categoryId
    : (expenseCategories[0]?.id ?? "")
  const selected = expenseCategories.find((c) => c.id === selectedId)

  const saving = createPlan.isPending || updatePlan.isPending

  function goStep2() {
    setNav("fwd")
    setStep(2)
  }

  function goStep1() {
    setNav("back")
    setStep(1)
  }

  function save() {
    const amountMinor = amountValueMinor(amount)

    if (amountMinor <= 0) {
      setStep(1)
      showToast("info", t("enterAmount"))

      return
    }

    if (!selectedId) {
      setStep(1)
      showToast("info", t("pickCategory"))

      return
    }

    // Empty title falls back to the category name — the backend needs a
    // non-empty one, and the row heading reads the same as a transaction's.
    const name = title.trim() || label(selected)

    const onSuccess = () => {
      closePlan()
      showToast("success", editing ? t("updated") : t("added"))
    }

    if (editing) {
      updatePlan.mutate(
        { id: plan.id, categoryId: selectedId, title: name, amountMinor },
        { onSuccess },
      )
    } else {
      createPlan.mutate(
        { month, categoryId: selectedId, title: name, amountMinor },
        { onSuccess },
      )
    }
  }

  return (
    <BottomSheet
      title={editing ? t("sheetTitleEdit") : t("sheetTitleNew")}
      onClose={closePlan}
      onBack={step === 2 ? goStep1 : undefined}
    >
      <div
        key={step}
        className="anim-step"
        style={
          { "--step-from": nav === "fwd" ? "16px" : "-16px" } as CSSProperties
        }
      >
        <div className="text-primary mt-3 mb-2 font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
          {step === 1 ? t("step1") : t("step2")}
        </div>

        {step === 1 ? (
          <>
            <div className="mt-1 mb-1.5">
              <AmountDisplay value={amount} />
            </div>
            <div className="text-faint mt-1 mb-2 font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
              {t("category")}
            </div>
            <div className="pb-3">
              {categories.isPending ? (
                <CategoryChipsSkeleton />
              ) : (
                <CategoryChips
                  categories={expenseCategories}
                  selectedId={selectedId}
                  onPick={setCategoryId}
                />
              )}
            </div>
            <Numpad value={amount} onChange={setAmount} />
            <button
              type="button"
              onClick={goStep2}
              className="border-border text-muted-foreground mt-3 flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed p-3 text-[13.5px] font-medium"
            >
              ＋ {title.trim() || t("addTitle")}
            </button>
            <button
              type="button"
              onClick={save}
              disabled={saving}
              className="bg-primary text-primary-foreground mt-2.5 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
            >
              {saving ? t("saving") : t("save")}
            </button>
          </>
        ) : (
          <>
            <div className="border-rowline mb-4 flex items-center justify-between gap-3 border-b pb-4">
              <div className="inline-flex items-baseline gap-1.5">
                <span
                  className="text-2xl font-bold tracking-[-0.02em]"
                  style={{
                    color: amount === "" ? "var(--faint)" : "var(--foreground)",
                  }}
                >
                  {amount === "" ? "0" : amount}
                </span>
                <span className="text-faint text-[13px] font-semibold">Br</span>
              </div>
              <span className="text-muted-foreground flex items-center gap-1.5 text-[12.5px] font-medium">
                <span
                  className="size-2 rounded-full"
                  style={{ background: selected?.color ?? "var(--faint)" }}
                />
                {label(selected)}
              </span>
            </div>
            <label
              htmlFor="plan-title"
              className="text-faint mb-2 block font-mono text-[10px] font-semibold tracking-[0.06em] uppercase"
            >
              {t("titleLabel")}
            </label>
            <input
              id="plan-title"
              autoFocus
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") save()
              }}
              placeholder={t("titlePlaceholder")}
              aria-label={t("titleLabel")}
              maxLength={60}
              className="text-foreground placeholder:text-faint border-primary bg-secondary focus:border-primary w-full rounded-2xl border-[1.5px] px-3.5 py-3 text-[14px] outline-none"
            />
            <button
              type="button"
              onClick={save}
              disabled={saving}
              className="bg-primary text-primary-foreground mt-4 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
            >
              {saving ? t("saving") : t("save")}
            </button>
          </>
        )}
      </div>
    </BottomSheet>
  )
}
