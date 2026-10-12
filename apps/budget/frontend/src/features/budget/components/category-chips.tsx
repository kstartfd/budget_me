"use client"

import { useEffect, useRef } from "react"
import { useTranslations } from "next-intl"

import { Skeleton } from "@/components/ui/skeleton"

import type { Category } from "../schemas"
import { wash, washStrong } from "./wash"

/**
 * Category chip row used in the sheets: color dot + name, the picked chip
 * tinted with its category hue. A swipeable strip on touch; with a mouse the
 * chips wrap and scroll vertically instead (see .chips in globals.css).
 */
export function CategoryChips({
  categories,
  selectedId,
  onPick,
  onAddNew,
}: {
  categories: Category[]
  selectedId: string
  onPick: (id: string) => void
  /** When set, a trailing "＋ Своя" chip lets the user add a custom category. */
  onAddNew?: () => void
}) {
  const t = useTranslations("categories")
  const tAdd = useTranslations("add")
  const selectedChip = useRef<HTMLButtonElement>(null)

  // Editing a row (or just having many categories) can leave the picked chip
  // out of view — off to the right on touch, below the fold on a mouse. Bring
  // it into the row's own viewport; "nearest" is a no-op when it already is.
  useEffect(() => {
    selectedChip.current?.scrollIntoView({ block: "nearest", inline: "nearest" })
  }, [selectedId])

  return (
    <div className="scr chips flex gap-2 overflow-x-auto pb-1">
      {categories.map((c) => {
        const on = c.id === selectedId
        return (
          <button
            key={c.id}
            ref={on ? selectedChip : undefined}
            type="button"
            aria-pressed={on}
            onClick={() => onPick(c.id)}
            className="flex shrink-0 items-center gap-[7px] rounded-full border px-[13px] py-2 text-[12.5px] font-medium whitespace-nowrap"
            style={
              on
                ? {
                    background: wash(c.color),
                    borderColor: washStrong(c.color),
                    color: "var(--foreground)",
                  }
                : {
                    background: "var(--secondary)",
                    borderColor: "transparent",
                    color: "var(--muted-foreground)",
                  }
            }
          >
            <span
              className="size-2 rounded-full"
              style={{ background: c.color }}
            />
            {c.slug ? t(c.slug) : (c.name ?? "—")}
          </button>
        )
      })}
      {onAddNew && (
        <button
          type="button"
          onClick={onAddNew}
          className="flex shrink-0 items-center gap-[6px] rounded-full border border-dashed px-[13px] py-2 text-[12.5px] font-medium whitespace-nowrap"
          style={{
            borderColor: "var(--primary-line)",
            color: "var(--primary)",
            background: "transparent",
          }}
        >
          ＋ {tAdd("custom")}
        </button>
      )}
    </div>
  )
}

/**
 * Placeholder chip row shown while the category query is still pending, so the
 * sheet keeps its height instead of collapsing to an empty gap.
 */
export function CategoryChipsSkeleton() {
  return (
    <div aria-hidden className="scr chips flex gap-2 overflow-x-auto pb-1">
      {[72, 88, 64, 96, 76].map((w, i) => (
        <Skeleton
          key={i}
          className="h-9 shrink-0 rounded-full"
          style={{ width: w }}
        />
      ))}
    </div>
  )
}
