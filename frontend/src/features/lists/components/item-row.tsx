"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { PencilIcon, TrashIcon } from "@/components/app-icons"

import type { Item } from "../schemas"
import { StarRating } from "./star-rating"
import { VerdictBadge } from "./verdict-badge"

/**
 * One item row (design PHONE «Аптечка»): title + purpose, verdict + stars,
 * edit/delete. Delete asks first — same explicit Отмена/Удалить swap as
 * tasks' TaskRow, for the same reason (no accidental taps in a scroll list).
 */
export function ItemRow({
  item,
  onEdit,
  onDelete,
}: {
  item: Item
  onEdit: () => void
  onDelete: () => void
}) {
  const t = useTranslations("lists")
  const [confirmingDelete, setConfirmingDelete] = useState(false)

  return (
    <div className="flex items-center gap-3 px-[15px] py-[13px]">
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

      {confirmingDelete ? (
        <div className="flex shrink-0 items-center gap-1.5">
          <button
            type="button"
            onClick={() => setConfirmingDelete(false)}
            className="border-border bg-secondary text-foreground rounded-full border px-2.5 py-[7px] text-[11.5px] font-semibold"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            autoFocus
            onClick={onDelete}
            className="bg-destructive rounded-full px-2.5 py-[7px] text-[11.5px] font-semibold text-white"
          >
            {t("confirmDelete")}
          </button>
        </div>
      ) : (
        <div className="flex shrink-0 items-center gap-1.5">
          <button
            type="button"
            aria-label={t("editAria", { name: item.title })}
            onClick={onEdit}
            className="border-border bg-secondary text-muted-foreground hover:text-foreground flex size-[30px] items-center justify-center rounded-full border transition-colors"
          >
            <PencilIcon />
          </button>
          <button
            type="button"
            aria-label={t("deleteAria", { name: item.title })}
            onClick={() => setConfirmingDelete(true)}
            className="border-border bg-secondary text-muted-foreground hover:text-destructive flex size-[30px] items-center justify-center rounded-full border transition-colors"
          >
            <TrashIcon />
          </button>
        </div>
      )}
    </div>
  )
}
