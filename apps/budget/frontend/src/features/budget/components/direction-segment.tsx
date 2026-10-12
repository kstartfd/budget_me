"use client"

import { useTranslations } from "next-intl"

import type { Direction } from "../schemas"

/**
 * Income / expense segmented control (design 3b · TWO STEP). Two pills in a
 * tinted track; the active one lifts onto a card surface — income tinted
 * green, expense in ink. Custom (not shadcn) to match the design exactly.
 */
export function DirectionSegment({
  value,
  onChange,
}: {
  value: Direction
  onChange: (next: Direction) => void
}) {
  const t = useTranslations("add")
  return (
    <div className="bg-secondary flex gap-1 rounded-xl p-[3px]">
      {(["expense", "income"] as const).map((dir) => {
        const on = value === dir
        return (
          <button
            key={dir}
            type="button"
            aria-pressed={on}
            onClick={() => onChange(dir)}
            className="flex-1 rounded-[10px] py-[9px] text-[12.5px] font-semibold transition-colors"
            style={
              on
                ? {
                    background: "var(--card)",
                    color:
                      dir === "income"
                        ? "var(--success)"
                        : "var(--foreground)",
                    boxShadow: "var(--shadow-card)",
                  }
                : { background: "transparent", color: "var(--muted-foreground)" }
            }
          >
            {t(dir)}
          </button>
        )
      })}
    </div>
  )
}
