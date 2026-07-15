"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { useSetBalance } from "../mutations"
import { AmountDisplay } from "./amount-display"
import { BottomSheet } from "./bottom-sheet"
import { Numpad, amountValueMinor } from "./numpad"
import { useBudgetUI } from "./ui-context"

/**
 * "Сколько у тебя сейчас денег" — the sheet that states the balance. Saving
 * re-anchors it: from this moment on, only entries made afterwards move it, so
 * the same sheet doubles as the correction when the figure drifts from reality.
 */
export function BalanceSheet() {
  const { balanceSheet } = useBudgetUI()

  if (!balanceSheet) return null

  return <BalanceSheetInner key={balanceSheet.prefill ?? "new"} />
}

function BalanceSheetInner() {
  const t = useTranslations("balance")
  const { balanceSheet, closeBalance, showToast } = useBudgetUI()
  const setBalance = useSetBalance()

  const [amount, setAmount] = useState(balanceSheet?.prefill ?? "")

  function save() {
    const balanceMinor = amountValueMinor(amount)

    if (amount.trim() === "") {
      showToast("info", t("enterAmount"))

      return
    }

    setBalance.mutate(balanceMinor, {
      onSuccess: () => {
        closeBalance()
        showToast("success", t("saved"))
      },
    })
  }

  return (
    <BottomSheet title={t("sheetTitle")} onClose={closeBalance}>
      <p className="text-muted-foreground mt-3 text-center text-[12.5px]">
        {t("sheetHint")}
      </p>

      <div className="mt-3 mb-1">
        <AmountDisplay value={amount} size={42} />
      </div>

      <Numpad value={amount} onChange={setAmount} />

      <button
        type="button"
        onClick={save}
        disabled={setBalance.isPending}
        className="bg-primary text-primary-foreground mt-3.5 w-full rounded-2xl p-4 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
      >
        {setBalance.isPending ? t("saving") : t("save")}
      </button>
    </BottomSheet>
  )
}
