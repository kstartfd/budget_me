"use client"

import { useTranslations } from "next-intl"

import { formatBr } from "@/lib/money"

import { useCategoryLabel } from "../category-label"
import { useCategories } from "../queries"
import { useDeleteTransaction } from "../mutations"
import { BottomSheet } from "./bottom-sheet"
import { useBudgetUI } from "./ui-context"

/**
 * Detail + actions for one transaction, opened by tapping its row (a reliable
 * touch entry point — the old long-press never fired in the Telegram webview).
 * The free-text comment lives in `title`; the row truncates it, so here it is
 * shown in full, wrapped, over the amount / category / date and Edit + Delete.
 * Global (mounted once in the shell) so both the dashboard and the transactions
 * list open it via `openDetail` without each wiring its own sheet.
 */
export function TxDetailSheet() {
  const t = useTranslations("transactions")
  const label = useCategoryLabel()
  const categories = useCategories()
  const { detailTx: tx, closeDetail, showToast, openEdit } = useBudgetUI()
  const deleteTx = useDeleteTransaction()

  if (!tx) return null

  const category = categories.data?.find((c) => c.id === tx.categoryId)
  const income = tx.direction === "income"

  function remove() {
    if (!tx) return

    deleteTx.mutate(tx.id, {
      onSuccess: () => {
        closeDetail()
        showToast("info", t("deleted", { title: tx.title }))
      },
      onError: () => showToast("danger", t("deleteFailed")),
    })
  }

  return (
    <BottomSheet title={t("detailTitle")} onClose={closeDetail}>
      <div className="mt-3 mb-4 text-center">
        <div className="text-[30px] font-bold tracking-[-0.02em]">
          {income ? "+" : "−"}
          {formatBr(tx.amountMinor)} Br
        </div>
        <div className="text-faint mt-1 text-[12.5px] font-medium">
          {label(category)} · {tx.date} · {tx.time}
        </div>
      </div>

      <div className="bg-secondary mb-5 rounded-2xl px-4 py-3">
        <div className="text-faint font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
          {t("commentLabel")}
        </div>
        <p className="mt-1 text-[14px] leading-[1.4] break-words whitespace-pre-wrap">
          {tx.title}
        </p>
      </div>

      <div className="flex flex-col gap-2.5">
        <button
          type="button"
          onClick={() => {
            closeDetail()
            openEdit(tx)
          }}
          className="bg-secondary text-foreground active:bg-border w-full rounded-2xl p-4 text-[15px] font-bold"
        >
          {t("edit")}
        </button>
        <button
          type="button"
          onClick={remove}
          disabled={deleteTx.isPending}
          className="bg-destructive/10 text-destructive active:bg-destructive/15 w-full rounded-2xl p-4 text-[15px] font-bold disabled:opacity-60"
        >
          {deleteTx.isPending ? t("deleting") : t("delete")}
        </button>
      </div>
    </BottomSheet>
  )
}
