"use client"

import { useTranslations } from "next-intl"

import { BottomSheet } from "./bottom-sheet"

/**
 * Plain-language explainer for the dashboard's money figures, opened by the ⓘ
 * on the hero card. «Остаток» (wallet) and «Свободно» (income − spend) read as
 * near-synonyms at a glance, so this spells out what each one actually means.
 */
export function MoneyInfoSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations("moneyInfo")

  const terms = ["balance", "income", "free"] as const

  return (
    <BottomSheet title={t("title")} onClose={onClose}>
      <div className="mt-2 mb-1 flex flex-col gap-4">
        {terms.map((k) => (
          <div key={k}>
            <div className="text-[14px] font-bold tracking-[-0.01em]">
              {t(`${k}Term`)}
            </div>
            <p className="text-muted-foreground mt-1 text-[13px] leading-[1.45]">
              {t(`${k}Desc`)}
            </p>
          </div>
        ))}
      </div>
    </BottomSheet>
  )
}
