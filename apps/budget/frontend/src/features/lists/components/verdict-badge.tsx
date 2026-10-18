"use client"

import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

import type { Verdict } from "../schemas"

const tone: Record<Verdict, string> = {
  works: "text-success bg-[color:var(--success-wash)]",
  meh: "text-warn bg-warn/10",
  no: "text-destructive bg-destructive/10",
  unknown: "text-faint bg-secondary",
}

/** Small colored chip for whether an item helped — works/meh/no/unknown. */
export function VerdictBadge({ verdict }: { verdict: Verdict }) {
  const t = useTranslations("lists")

  return (
    <span
      className={cn(
        "shrink-0 rounded-full px-2.5 py-[3px] text-[11px] font-semibold tracking-[0.04em] uppercase",
        tone[verdict],
      )}
    >
      {t(`verdict.${verdict}`)}
    </span>
  )
}
