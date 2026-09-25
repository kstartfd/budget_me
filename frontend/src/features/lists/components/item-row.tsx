"use client"

import { useTranslations } from "next-intl"

import { ChevronRightIcon } from "@/components/app-icons"

import type { Item } from "../schemas"
import { StarRating } from "./star-rating"
import { VerdictBadge } from "./verdict-badge"

/** A compact list entry; the full item, including its photo, opens on tap. */
export function ItemRow({ item, onOpen }: { item: Item; onOpen: () => void }) {
  const t = useTranslations("lists")

  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={t("viewItem", { name: item.title })}
      className="hover:bg-secondary/60 focus-visible:ring-primary flex w-full items-center gap-3 px-[15px] py-[13px] text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-inset"
    >
      <div className="min-w-0 flex-1">
        <div className="truncate text-[14.5px] font-medium">{item.title}</div>
        <div className="mt-1 flex items-center gap-2">
          {item.purpose && (
            <span className="text-muted-foreground truncate text-[12px]">
              {item.purpose}
            </span>
          )}
          <VerdictBadge verdict={item.verdict} />
        </div>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1.5">
        <StarRating rating={item.rating} />
        <span className="text-faint font-mono text-[10.5px]">
          {item.rating ? `${item.rating}/10` : "—/10"}
        </span>
      </div>
      <span aria-hidden className="text-faint shrink-0">
        <ChevronRightIcon />
      </span>
    </button>
  )
}
