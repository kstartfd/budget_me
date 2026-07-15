"use client"

import { useTranslations } from "next-intl"

import { SparkFillIcon } from "@/components/app-icons"
import { useCategoryLabel } from "@/features/budget/category-label"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { wash } from "@/features/budget/components/wash"
import { useCategories, useInsights, type Insight } from "@/features/budget"
import { formatBr } from "@/lib/money"
import { AiSkeleton } from "./skeleton"

const toneColor: Record<string, string> = {
  info: "#5b8def",
  warn: "#e0a33e",
  success: "#35b98f",
  danger: "#ea6a6a",
}

/**
 * "AI-советы". Two layers, and the difference matters:
 *
 * - The cards are FACTS the backend computed from the user's own ledger and
 *   limits — every figure is carried in the payload. (They replaced cards whose
 *   numbers were written into the translation files and were pure fiction.)
 * - The advice on top is a language model phrasing those same facts. It decides
 *   nothing and invents no number; it is absent until written, because it takes
 *   seconds and no screen waits for it.
 */
export default function AiPage() {
  const t = useTranslations("ai")
  const { openLimit, fx } = useBudgetUI()
  const label = useCategoryLabel()
  const insights = useInsights()
  const categories = useCategories()

  if (insights.isPending) return <AiSkeleton />

  const cards = insights.data?.insights ?? []
  const advice = insights.data?.advice ?? null
  const overspendMinor = insights.data?.overspendMinor ?? 0
  const overspendApprox = fx(overspendMinor)
  const catById = new Map((categories.data ?? []).map((c) => [c.id, c]))

  function categoryName(card: Insight): string {
    const category = card.categoryId ? catById.get(card.categoryId) : undefined

    return category ? label(category) : ""
  }

  return (
    <div className="anim-fade flex flex-col gap-4">
      <h1 className="px-0.5 text-[22px] font-bold tracking-[-0.02em]">
        {t("title")}
      </h1>

      {advice && (
        <section className="bg-card border-[color:var(--primary-line)] rounded-[22px] border p-5 shadow-[var(--shadow-card)]">
          <div className="flex items-center gap-2.5">
            <div className="bg-primary text-primary-foreground flex size-8 shrink-0 items-center justify-center rounded-[10px]">
              <SparkFillIcon />
            </div>
            <span className="text-primary font-mono text-[10px] font-semibold tracking-[0.08em] uppercase">
              {t("adviceKicker")}
            </span>
          </div>
          <p className="mt-3 text-[14px] leading-relaxed">{advice}</p>
        </section>
      )}

      {overspendMinor > 0 && (
        <section className="bg-card border-[color:var(--destructive)]/30 rounded-[22px] border p-5 shadow-[var(--shadow-card)]">
          <div className="text-destructive font-mono text-[10.5px] font-semibold tracking-[0.08em] uppercase">
            {t("overspend")}
          </div>
          <div className="mt-2.5 flex items-baseline gap-[9px]">
            <span className="text-[32px] font-bold tracking-[-0.03em]">
              {formatBr(overspendMinor)}
            </span>
            <span className="text-faint text-sm font-semibold">Br</span>
          </div>
          <div className="text-muted-foreground mt-1 font-mono text-[12.5px] font-medium">
            {overspendApprox && <>≈ {overspendApprox} · </>}
            {t("overspendHint")}
          </div>
        </section>
      )}

      {cards.length === 0 && !advice && (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {t("noInsights")}
        </p>
      )}

      {cards.map((card, i) => {
        const color = toneColor[card.tone]
        const name = categoryName(card)
        const spent = formatBr(card.spentMinor ?? 0)
        const limit = formatBr(card.limitMinor ?? 0)
        const over = formatBr((card.spentMinor ?? 0) - (card.limitMinor ?? 0))
        const suggest = card.suggestMinor ?? 0

        return (
          <section
            key={`${card.kind}:${card.categoryId ?? i}`}
            className="bg-card border-border flex gap-[13px] rounded-[20px] border p-[17px] shadow-[var(--shadow-card)]"
          >
            <span
              className="flex size-[34px] shrink-0 items-center justify-center rounded-[11px] text-[15px] font-bold"
              style={{ background: wash(color), color }}
            >
              {card.kind === "overspend" ? "↗" : card.kind === "onTrack" ? "✓" : "◎"}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-[14.5px] font-semibold tracking-[-0.01em]">
                {t(`card.${card.kind}.title`, { name })}
              </div>
              <div className="text-muted-foreground mt-[5px] text-[13px] leading-[1.55]">
                {t(`card.${card.kind}.body`, { name, spent, limit, over })}
              </div>

              {card.kind === "overspend" && card.categoryId && (
                <button
                  type="button"
                  onClick={() => openLimit(card.categoryId!, card.limitMinor)}
                  className="text-primary mt-[11px] text-[12.5px] font-semibold"
                >
                  {t("card.overspend.action")}
                </button>
              )}

              {card.kind === "noLimit" && card.categoryId && suggest > 0 && (
                <button
                  type="button"
                  onClick={() => openLimit(card.categoryId!, suggest)}
                  className="bg-primary text-primary-foreground mt-[11px] rounded-[11px] px-3.5 py-2 text-[12.5px] font-semibold"
                >
                  {t("card.noLimit.action", { amount: formatBr(suggest) })}
                </button>
              )}
            </div>
          </section>
        )
      })}
    </div>
  )
}
