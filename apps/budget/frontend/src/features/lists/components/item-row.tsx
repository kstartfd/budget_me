"use client"

import { useTranslations } from "next-intl"

import { ChevronRightIcon } from "@/components/app-icons"
import { cn } from "@/lib/utils"

import type { Item } from "../schemas"
import { StarRating } from "./star-rating"
import { VerdictBadge } from "./verdict-badge"

/**
 * A compact list entry; the full item, including its photo, opens on tap.
 * A card of its own on a phone (name + rating on top, note + verdict under),
 * a table row on desktop (name / note on the left, rating, chevron) — the
 * same grid cells, placed differently by breakpoint.
 */
export function ItemRow({ item, onOpen }: { item: Item; onOpen: () => void }) {
  const t = useTranslations("lists")

  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={t("viewItem", { name: item.title })}
      className={cn(
        "bg-card border-border grid w-full grid-cols-[minmax(0,1fr)_auto] items-start gap-x-3 gap-y-2.5 rounded-2xl border p-3.5 text-left outline-none transition-colors",
        "focus-visible:ring-primary focus-visible:ring-2 focus-visible:ring-inset",
        "lg:grid-cols-[minmax(0,1fr)_auto_16px] lg:items-center lg:gap-x-5 lg:gap-y-1.5 lg:rounded-none lg:border-0 lg:border-b lg:px-[22px] lg:py-4 lg:last:border-b-0 lg:hover:bg-secondary/60",
      )}
    >
      <span className="col-start-1 row-start-1 truncate text-[15px] font-semibold">
        {item.title}
      </span>

      <span className="col-start-2 row-start-1 flex flex-col items-end gap-0.5 lg:row-span-2">
        <StarRating rating={item.rating} size={15} />
        <span className="text-faint font-mono text-[11px] lg:text-xs">
          {item.rating ? `${item.rating}/10` : "—/10"}
        </span>
      </span>

      <span className="col-span-2 row-start-2 flex min-w-0 flex-wrap items-center gap-2 lg:col-span-1 lg:col-start-1 lg:gap-2.5">
        {item.purpose && (
          <span className="text-muted-foreground truncate text-[13px]">
            {item.purpose}
          </span>
        )}
        <VerdictBadge verdict={item.verdict} />
      </span>

      <span
        aria-hidden
        className="text-faint hidden lg:col-start-3 lg:row-span-2 lg:row-start-1 lg:block"
      >
        <ChevronRightIcon />
      </span>
    </button>
  )
}
