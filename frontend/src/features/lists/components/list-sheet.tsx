"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { CATEGORY_PALETTE, DEFAULT_CATEGORY_COLOR } from "@/features/categories/palette"
import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { cn } from "@/lib/utils"

import { useCreateList } from "../mutations"

/**
 * Create a personal list (Витамины, Уход за телом, whatever). Reuses the
 * categories colour palette — same swatch set, no reason to invent a second
 * one for what is visually the same kind of coloured dictionary row.
 */
export function ListSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations("lists")
  const tColors = useTranslations("categoriesPage")
  const create = useCreateList()

  const [name, setName] = useState("")
  const [color, setColor] = useState<string>(DEFAULT_CATEGORY_COLOR)

  const canSubmit = name.trim().length > 0 && !create.isPending

  function submit() {
    if (!canSubmit) return

    create.mutate({ name: name.trim(), color }, { onSuccess: onClose })
  }

  return (
    <BottomSheet title={t("newListTitle")} onClose={onClose}>
      <div className="flex flex-col gap-3 pt-4">
        <input
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("listNamePlaceholder")}
          maxLength={60}
          className="border-primary bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3.5 text-[15px] outline-none"
        />

        <div
          role="radiogroup"
          aria-label={t("listColorLabel")}
          className="flex flex-wrap gap-3 px-0.5 py-1"
        >
          {CATEGORY_PALETTE.map((c) => {
            const on = color.toLowerCase() === c.hex.toLowerCase()
            return (
              <button
                key={c.hex}
                type="button"
                role="radio"
                aria-checked={on}
                aria-label={tColors(`colors.${c.key}`)}
                onClick={() => setColor(c.hex)}
                className={cn(
                  "size-7 rounded-full outline-2 outline-offset-2 outline-transparent",
                  on && "outline-foreground",
                )}
                style={{
                  background: c.hex,
                  boxShadow: "inset 0 0 0 1px rgba(255,255,255,0.18)",
                }}
              />
            )
          })}
        </div>

        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="bg-primary text-primary-foreground mt-1 w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-40"
        >
          {t("createList")}
        </button>
      </div>
    </BottomSheet>
  )
}
