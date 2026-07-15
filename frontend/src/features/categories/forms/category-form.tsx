"use client"

import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

import { CATEGORY_PALETTE } from "../palette"
import { categoryFormSchema, type CategoryFormValues } from "../schemas"

/**
 * Add/edit category form (design "add/edit sheet"): a name field and the
 * palette colour picker, with Save / Cancel and (edit only) a delete link.
 * Presentation-only — RHF+Zod own validation, the parent sheet owns the
 * mutations and the busy/close wiring. Rendered inside either a bottom sheet
 * (mobile) or a centered dialog (desktop); layout is identical in both.
 */
export function CategoryForm({
  mode,
  defaultValues,
  submitting,
  deleting,
  onSubmit,
  onDelete,
  onCancel,
}: {
  mode: "create" | "edit"
  defaultValues: CategoryFormValues
  submitting: boolean
  deleting: boolean
  onSubmit: (values: CategoryFormValues) => void
  onDelete?: () => void
  onCancel: () => void
}) {
  const t = useTranslations("categoriesPage")
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<CategoryFormValues>({
    resolver: zodResolver(categoryFormSchema),
    defaultValues,
  })

  const busy = submitting || deleting

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="mt-1">
      <div className="text-primary mt-4 mb-[9px] text-[11px] font-bold tracking-[0.06em] uppercase">
        {t("nameLabel")}
      </div>
      <Input
        autoFocus
        aria-label={t("nameLabel")}
        aria-invalid={errors.name ? true : undefined}
        placeholder={t("namePlaceholder")}
        maxLength={40}
        className="bg-secondary border-border text-foreground focus-visible:border-primary h-auto rounded-[14px] px-3.5 py-3 text-sm focus-visible:ring-0"
        {...register("name")}
      />

      <div className="text-primary mt-5 mb-[9px] text-[11px] font-bold tracking-[0.06em] uppercase">
        {t("colorLabel")}
      </div>
      <Controller
        control={control}
        name="color"
        render={({ field }) => (
          <div
            role="radiogroup"
            aria-label={t("colorGroup")}
            className="flex flex-wrap gap-3"
          >
            {CATEGORY_PALETTE.map((c) => {
              const on = field.value.toLowerCase() === c.hex.toLowerCase()
              return (
                <button
                  key={c.hex}
                  type="button"
                  role="radio"
                  aria-checked={on}
                  aria-label={t(`colors.${c.key}`)}
                  onClick={() => field.onChange(c.hex)}
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
        )}
      />

      <div className="mt-[22px] flex gap-2.5">
        <Button
          type="submit"
          disabled={busy}
          className="text-primary-foreground h-auto flex-1 rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
        >
          {submitting ? t("saving") : t("save")}
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={onCancel}
          disabled={busy}
          className="text-muted-foreground h-auto rounded-[18px] px-5 py-[15px] text-sm font-semibold"
        >
          {t("cancel")}
        </Button>
      </div>

      {mode === "edit" && onDelete && (
        <button
          type="button"
          onClick={onDelete}
          disabled={busy}
          className="text-destructive mt-3.5 block w-full p-1 text-center text-[13px] font-semibold disabled:opacity-60"
        >
          {deleting ? t("deleting") : t("delete")}
        </button>
      )}
    </form>
  )
}
