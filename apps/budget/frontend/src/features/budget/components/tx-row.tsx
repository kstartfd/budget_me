"use client"

import { useTranslations } from "next-intl"

import { formatBr } from "@/lib/money"
import { cn } from "@/lib/utils"

import { useCategoryLabel } from "../category-label"
import type { Category, Transaction } from "../schemas"
import { useBudgetUI } from "./ui-context"
import { wash } from "./wash"

/**
 * One transaction row (dashboard "recent" and the transactions screen): tinted
 * square with the category dot, title over "category · time", mono amount with
 * its ≈USD line. Direction styles the amount: income is "+" tinted the success
 * green (matching amount-display), expense is "−" in the normal foreground.
 *
 * The whole row is a button: tapping it opens the transaction detail sheet
 * (full comment + Edit/Delete). It replaced a "⋯" button, which itself replaced
 * an onContextMenu long-press that never fired in the Telegram webview — so the
 * row's own tap is now the single, reliable entry point on a phone.
 */
export function TxRow({
  tx,
  category,
  meta,
}: {
  tx: Transaction
  category: Category | undefined
  /** Right part of the subtitle: time on the list screen, day on recent. */
  meta: string
}) {
  const t = useTranslations("transactions")
  const label = useCategoryLabel()
  const { fx, openDetail } = useBudgetUI()
  const approx = fx(tx.amountMinor)
  const color = category?.color ?? "#9198a6"
  const income = tx.direction === "income"

  return (
    <button
      type="button"
      onClick={() => openDetail(tx)}
      className="hover:bg-secondary/50 active:bg-secondary flex w-full items-center gap-3 px-4 py-3 text-left transition-colors"
    >
      <span
        className="flex size-9 shrink-0 items-center justify-center rounded-[11px]"
        style={{ background: wash(color) }}
      >
        <span className="size-[9px] rounded-full" style={{ background: color }} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium">{tx.title}</div>
        <div className="text-faint mt-0.5 text-xs">
          {label(category)} · {meta}
          {!tx.counted && <span className="italic"> · {t("uncounted")}</span>}
        </div>
      </div>
      <div className="text-right">
        <div
          className={cn(
            "font-mono text-sm font-semibold",
            income && "text-success",
          )}
        >
          {income ? "+" : "−"}
          {formatBr(tx.amountMinor)} Br
        </div>
        {approx && (
          <div
            className={cn(
              "mt-0.5 font-mono text-[11px]",
              income ? "text-success/70" : "text-faint",
            )}
          >
            {approx}
          </div>
        )}
      </div>
    </button>
  )
}
