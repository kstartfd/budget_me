"use client"

import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

import type { Verdict } from "../schemas"

const tone: Record<Verdict, string> = {
  works: "text-success bg-[color:var(--success-wash)] border-[color:var(--success-line)]",
  meh: "text-warn bg-warn/10 border-warn/30",
  no: "text-destructive bg-destructive/10 border-destructive/30",
  unknown: "text-faint bg-secondary border-border",
}

/** Small colored chip for whether an item helped — works/meh/no/unknown. */
export function VerdictBadge({ verdict }: { verdict: Verdict }) {
  const t = useTranslations("lists")

  return (
    <span
      className={cn(
        "shrink-0 rounded-full border px-2 py-[3px] text-[10px] font-bold tracking-[0.03em] uppercase",
        tone[verdict],
      )}
    >
      {t(`verdict.${verdict}`)}
    </span>
  )
}
