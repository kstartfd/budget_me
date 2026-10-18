"use client"

import { useState, type CSSProperties } from "react"
import { useLocale, useTranslations } from "next-intl"

import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { useToday } from "@/features/settings"
import { formatBr } from "@/lib/money"
import { dayLabel, nowTime } from "@/lib/month"

import { useCategoryLabel } from "../category-label"
import { useAddTransaction, useUpdateTransaction } from "../mutations"
import { useCategories } from "../queries"
import type { Direction, Transaction } from "../schemas"
import { AmountDisplay } from "./amount-display"
import { BottomSheet } from "./bottom-sheet"
import { CategoryChips, CategoryChipsSkeleton } from "./category-chips"
import { DirectionSegment } from "./direction-segment"
import { Numpad, amountValueMinor } from "./numpad"
import { useBudgetUI } from "./ui-context"

/**
 * "Новая операция" / "Изменить операцию" — the design's two-step add flow
 * (3b · TWO STEP), reused for editing.
 *
 * Step 1 (amount): income/expense segment, the custom numpad, and category
 * chips. The primary path is short — «Сохранить» saves immediately (note
 * optional). «＋ Комментарий» advances to step 2. No text input lives here, so
 * the OS keyboard never appears.
 *
 * Step 2 (note): the free-text note (the only place the OS keyboard shows) with
 * the amount as a small summary and a way back. Splitting the steps keeps the
 * custom numpad and the OS keyboard from ever coexisting.
 *
 * Opened either for a new row (addOpen) or to edit an existing one (editTx). The
 * inner is unmounted when the sheet closes, so it re-initialises from `editTx`
 * on every open — no manual reset needed.
 */
export function AddSheet() {
  const { addOpen, editTx } = useBudgetUI()

  if (!addOpen && !editTx) return null

  return <AddSheetInner key={editTx?.id ?? "new"} editTx={editTx} />
}

function AddSheetInner({ editTx }: { editTx: Transaction | null }) {
  const t = useTranslations("add")
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const label = useCategoryLabel()
  const { closeAdd, showToast } = useBudgetUI()
  const categories = useCategories()
  const addTransaction = useAddTransaction()
  const updateTransaction = useUpdateTransaction()
  const today = useToday()

  const editing = editTx !== null
  const [direction, setDirection] = useState<Direction>(
    editTx?.direction ?? "expense",
  )
  const [amount, setAmount] = useState(
    editTx ? formatBr(editTx.amountMinor) : "",
  )
  const [categoryId, setCategoryId] = useState(editTx?.categoryId ?? "")
  const [note, setNote] = useState(editTx?.title ?? "")
  const [date, setDate] = useState(editTx?.date ?? today)
  const [counted, setCounted] = useState(editTx?.counted ?? true)
  const [step, setStep] = useState<1 | 2>(1)
  const [nav, setNav] = useState<"fwd" | "back">("fwd")

  // Categories shown depend on the segment: the real dictionary carries both
  // directions, so filter it on `direction`. The selection resolves against the
  // active list so a stale pick (or none yet) falls back to its first item.
  const activeCategories = (categories.data ?? []).filter(
    (c) => c.direction === direction,
  )
  const activeCategoriesPending = categories.isPending
  const selectedId = activeCategories.some((c) => c.id === categoryId)
    ? categoryId
    : (activeCategories[0]?.id ?? "")

  function close() {
    closeAdd()
  }

  function pickDirection(next: Direction) {
    setDirection(next)
    // Drop the pick so it resolves to the new list's first item.
    setCategoryId("")
  }

  function goStep2() {
    setNav("fwd")
    setStep(2)
  }

  function goStep1() {
    setNav("back")
    setStep(1)
  }

  const saving = addTransaction.isPending || updateTransaction.isPending

  function save() {
    const amountMinor = amountValueMinor(amount)
    if (amountMinor <= 0) {
      setStep(1)
      showToast("info", t("enterAmount"))
      return
    }
    const category = activeCategories.find((c) => c.id === selectedId)
    // Free-form note is the row title; fall back to the category name when empty.
    const title = note.trim() || label(category)
    // Only an expense can be kept out of the calculation; income always counts.
    const inCalculation = direction === "income" || counted

    if (editing) {
      // The time is preserved; the date can be changed via the picker.
      updateTransaction.mutate(
        {
          id: editTx.id,
          amountMinor,
          categoryId: selectedId,
          title,
          date,
          time: editTx.time,
          direction,
          counted: inCalculation,
        },
        {
          onSuccess: () => {
            close()
            showToast("success", t("updated"))
          },
        },
      )
      return
    }

    addTransaction.mutate(
      {
        amountMinor,
        categoryId: selectedId,
        title,
        date,
        time: nowTime(),
        direction,
        counted: inCalculation,
      },
      {
        onSuccess: () => {
          close()
          showToast(
            "success",
            direction === "income" ? t("savedIncome") : t("saved"),
          )
        },
      },
    )
  }

  const sign = direction === "income" ? "+" : "−"
  const amountColor =
    amount === ""
      ? "var(--faint)"
      : direction === "income"
        ? "var(--success)"
        : "var(--foreground)"
  const saveDisabled = saving
  const saveBg =
    direction === "income" ? "var(--success)" : "var(--primary)"

  return (
    <BottomSheet
      title={editing ? t("titleEdit") : t("title")}
      onClose={close}
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
            <DirectionSegment value={direction} onChange={pickDirection} />
            <div className="mt-4 mb-1.5">
              <AmountDisplay value={amount} direction={direction} />
            </div>
            <div className="mt-1 mb-2 flex items-center justify-between gap-2">
              <span className="text-faint font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
                {direction === "income" ? t("source") : t("category")}
              </span>
              {/* Spent-on date. Defaults to today, capped at today; a transparent
                  native date input over the pill opens the OS picker on tap. */}
              <div className="relative shrink-0">
                <div className="border-border bg-secondary text-foreground flex items-center gap-1.5 rounded-full border px-3 py-[5px] text-[12px] font-medium">
                  <svg
                    width="13"
                    height="13"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden
                  >
                    <rect x="3" y="4" width="18" height="18" rx="2" />
                    <path d="M16 2v4M8 2v4M3 10h18" />
                  </svg>
                  {dayLabel(date, locale, tCommon("today"), tCommon("yesterday"), today)}
                </div>
                <input
                  type="date"
                  value={date}
                  max={today}
                  onChange={(e) => e.target.value && setDate(e.target.value)}
                  aria-label={t("date")}
                  className="absolute inset-0 cursor-pointer opacity-0"
                />
              </div>
            </div>
            <div className="pb-3">
              {activeCategoriesPending ? (
                <CategoryChipsSkeleton />
              ) : (
                <CategoryChips
                  categories={activeCategories}
                  selectedId={selectedId}
                  onPick={setCategoryId}
                />
              )}
            </div>
            {direction === "expense" && (
              <div className="mb-3 flex items-center gap-2.5">
                <Checkbox
                  id="add-counted"
                  checked={counted}
                  onCheckedChange={(v) => setCounted(v === true)}
                />
                <Label
                  htmlFor="add-counted"
                  className="cursor-pointer text-[13px] font-medium"
                >
                  {t("counted")}
                </Label>
                {!counted && (
                  <span className="text-faint ml-auto truncate text-[11.5px]">
                    {t("countedOffHint")}
                  </span>
                )}
              </div>
            )}
            <Numpad value={amount} onChange={setAmount} />
            <button
              type="button"
              onClick={goStep2}
              className="border-border text-muted-foreground mt-3 flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed p-3 text-[13.5px] font-medium"
            >
              ＋ {t("addNote")}
            </button>
            <button
              type="button"
              onClick={save}
              disabled={saveDisabled}
              className="text-primary-foreground mt-2.5 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
              style={{ background: saveBg }}
            >
              {saveDisabled ? t("saving") : t("save")}
            </button>
          </>
        ) : (
          <>
            <div className="border-rowline mb-4 flex items-center justify-between gap-3 border-b pb-4">
              <div className="inline-flex items-baseline gap-1.5">
                <span
                  className="text-2xl font-bold tracking-[-0.02em]"
                  style={{ color: amountColor }}
                >
                  {sign}
                  {amount === "" ? "0" : amount}
                </span>
                <span className="text-faint text-[13px] font-semibold">Br</span>
              </div>
              <span className="text-muted-foreground flex items-center gap-1.5 text-[12.5px] font-medium">
                <span
                  className="size-2 rounded-full"
                  style={{
                    background:
                      activeCategories.find((c) => c.id === selectedId)?.color ??
                      "var(--faint)",
                  }}
                />
                {label(activeCategories.find((c) => c.id === selectedId))}
              </span>
            </div>
            <label
              htmlFor="add-note"
              className="text-faint mb-2 block font-mono text-[10px] font-semibold tracking-[0.06em] uppercase"
            >
              {t("note")}
            </label>
            <input
              id="add-note"
              autoFocus
              value={note}
              onChange={(e) => setNote(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") save()
              }}
              placeholder={t("notePlaceholder")}
              aria-label={t("note")}
              maxLength={60}
              className="text-foreground placeholder:text-faint border-primary bg-secondary focus:border-primary w-full rounded-2xl border-[1.5px] px-3.5 py-3 text-[14px] outline-none"
            />
            <button
              type="button"
              onClick={save}
              disabled={saveDisabled}
              className="text-primary-foreground mt-4 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
              style={{ background: saveBg }}
            >
              {saveDisabled ? t("saving") : t("save")}
            </button>
          </>
        )}
      </div>
    </BottomSheet>
  )
}
