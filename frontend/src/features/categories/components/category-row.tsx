"use client"

import { useTranslations } from "next-intl"

import { LockIcon, PencilIcon, TrashIcon } from "@/components/app-icons"
import type { Category } from "@/features/budget"

/**
 * One category row: colour swatch + name, then actions. Preset (shared)
 * categories are locked — a lock icon, no edit/delete. Personal categories
 * carry a «личная» tag plus edit and delete buttons.
 */
export function CategoryRow({
  category,
  name,
  isPreset,
  onEdit,
  onDelete,
}: {
  category: Category
  name: string
  isPreset: boolean
  onEdit: () => void
  onDelete: () => void
}) {
  const t = useTranslations("categoriesPage")

  return (
    <div className="flex items-center gap-[13px] px-[15px] py-[13px]">
      <span
        className="size-[30px] shrink-0 rounded-full"
        style={{
          background: category.color,
          boxShadow:
            "inset 0 0 0 1px rgba(255,255,255,0.18), 0 1px 3px rgba(0,0,0,0.18)",
        }}
      />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[14.5px] font-medium">{name}</div>
      </div>

      {!isPreset && (
        <span className="text-primary bg-[color:var(--primary-wash)] border-[color:var(--primary-line)] shrink-0 rounded-full border px-2 py-[3px] text-[10px] font-bold tracking-[0.05em] uppercase">
          {t("personal")}
        </span>
      )}

      <div className="flex shrink-0 items-center gap-1.5">
        {isPreset ? (
          <span
            className="text-faint flex size-[30px] items-center justify-center"
            title={t("presetLocked")}
            aria-label={t("presetLocked")}
          >
            <LockIcon />
          </span>
        ) : (
          <>
            <button
              type="button"
              aria-label={t("editAria", { name })}
              onClick={onEdit}
              className="border-border bg-secondary text-muted-foreground hover:text-foreground flex size-[30px] items-center justify-center rounded-full border transition-colors"
            >
              <PencilIcon />
            </button>
            <button
              type="button"
              aria-label={t("deleteAria", { name })}
              onClick={onDelete}
              className="border-border bg-secondary text-muted-foreground hover:text-destructive flex size-[30px] items-center justify-center rounded-full border transition-colors"
            >
              <TrashIcon />
            </button>
          </>
        )}
      </div>
    </div>
  )
}
