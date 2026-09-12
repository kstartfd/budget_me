"use client"

import type { ComponentType } from "react"
import { useTranslations } from "next-intl"

/**
 * Placeholder for a «Жизнь» section whose screens are not built yet. Deliberately
 * plain: it states what the section will hold and nothing else, so it never
 * reads as a broken screen — and it claims no add action, which is what leaves
 * the nav's ＋ disabled here.
 */
export function Soon({
  icon: Icon,
  title,
  description,
}: {
  icon: ComponentType<{ size?: number }>
  title: string
  description: string
}) {
  const t = useTranslations("soon")

  return (
    <div className="anim-fade flex flex-col gap-[18px] lg:max-w-2xl">
      <h1 className="text-[22px] font-bold tracking-[-0.02em]">{title}</h1>

      <div className="bg-card border-border flex flex-col items-center gap-3 rounded-[22px] border px-6 py-10 text-center shadow-[var(--shadow-card)]">
        <span className="bg-secondary text-faint flex size-[52px] items-center justify-center rounded-full">
          <Icon size={24} />
        </span>
        <span className="text-primary bg-[color:var(--primary-wash)] border-[color:var(--primary-line)] rounded-full border px-2.5 py-[3px] text-[10px] font-bold tracking-[0.05em] uppercase">
          {t("badge")}
        </span>
        <p className="text-muted-foreground max-w-[38ch] text-[13.5px]">
          {description}
        </p>
      </div>
    </div>
  )
}
