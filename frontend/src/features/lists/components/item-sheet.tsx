"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { cn } from "@/lib/utils"

import { useCreateItem, useUpdateItem } from "../mutations"
import type { Item, Verdict } from "../schemas"
import { StarRating } from "./star-rating"

const verdicts: Verdict[] = ["unknown", "works", "meh", "no"]

/**
 * Add-or-edit an item. `item` present → edit (PUT), absent → create (POST)
 * into `listId` — same dual-purpose shape as tasks' AddTaskSheet.
 */
export function ItemSheet({
  listId,
  item,
  onClose,
}: {
  listId: string
  item?: Item
  onClose: () => void
}) {
  const t = useTranslations("lists")
  const create = useCreateItem(listId)
  const update = useUpdateItem(listId)

  const [title, setTitle] = useState(item?.title ?? "")
  const [purpose, setPurpose] = useState(item?.purpose ?? "")
  const [verdict, setVerdict] = useState<Verdict>(item?.verdict ?? "unknown")
  const [rating, setRating] = useState<number | null>(item?.rating ?? null)
  const [note, setNote] = useState(item?.note ?? "")

  const pending = create.isPending || update.isPending
  const canSubmit = title.trim().length > 0 && !pending

  function submit() {
    if (!canSubmit) return

    const shared = {
      title: title.trim(),
      purpose: purpose.trim() || null,
      verdict,
      rating,
      note: note.trim() || null,
    }

    if (item) {
      update.mutate({ id: item.id, ...shared }, { onSuccess: onClose })

      return
    }

    create.mutate({ listId, ...shared }, { onSuccess: onClose })
  }

  return (
    <BottomSheet title={item ? t("editItemTitle") : t("addItemTitle")} onClose={onClose}>
      <div className="flex flex-col gap-3 pt-4">
        <input
          autoFocus
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("titlePlaceholder")}
          maxLength={200}
          className="border-primary bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3.5 text-[15px] outline-none"
        />
        <input
          value={purpose}
          onChange={(e) => setPurpose(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("purposePlaceholder")}
          maxLength={200}
          className="border-border bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3 text-[14px] outline-none"
        />
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("notePlaceholder")}
          maxLength={500}
          className="border-border bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3 text-[14px] outline-none"
        />

        <div>
          <div className="text-faint mb-1.5 px-0.5 font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {t("verdictLabel")}
          </div>
          <div className="bg-secondary flex gap-1 rounded-2xl p-1">
            {verdicts.map((v) => {
              const on = v === verdict
              return (
                <button
                  key={v}
                  type="button"
                  aria-pressed={on}
                  onClick={() => setVerdict(v)}
                  className={cn(
                    "flex-1 rounded-[13px] py-[9px] text-[12.5px] font-semibold transition-colors",
                    on ? "bg-card text-foreground shadow-[var(--shadow-card)]" : "text-faint",
                  )}
                >
                  {t(`verdict.${v}`)}
                </button>
              )
            })}
          </div>
        </div>

        <div className="flex items-center justify-between px-0.5">
          <span className="text-faint font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {t("ratingLabel")}
          </span>
          <div className="flex items-center gap-2">
            <StarRating rating={rating} size={26} onChange={setRating} />
            <span className="text-muted-foreground w-[38px] font-mono text-[12px]">
              {rating ? `${rating}/10` : "—/10"}
            </span>
            {rating != null && (
              <button
                type="button"
                onClick={() => setRating(null)}
                className="text-faint text-[11px] font-semibold"
              >
                {t("clearRating")}
              </button>
            )}
          </div>
        </div>

        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="bg-primary text-primary-foreground mt-1 w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-40"
        >
          {item ? t("editItemSubmit") : t("addItemSubmit")}
        </button>
      </div>
    </BottomSheet>
  )
}
