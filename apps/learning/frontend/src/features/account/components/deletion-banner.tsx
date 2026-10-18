"use client"

import { useFormatter, useTranslations } from "next-intl"

import { useAuth } from "@/features/auth"

import { useCancelAccountDeletion } from "../mutations"

/** Shown on every screen while a deletion is pending: one tap to call it off. */
export function DeletionBanner() {
  const t = useTranslations("account")
  const format = useFormatter()
  const { user } = useAuth()
  const cancel = useCancelAccountDeletion()

  const deleteAt = user?.delete_at
  if (!deleteAt) return null

  const date = format.dateTime(new Date(deleteAt), {
    day: "numeric",
    month: "long",
  })

  return (
    <div
      role="status"
      className="flex items-center gap-3 rounded-[16px] border border-[color:var(--destructive)]/30 bg-[color:var(--destructive)]/10 px-3.5 py-3"
    >
      <span className="text-destructive min-w-0 flex-1 text-[12.5px] font-medium">
        {t("bannerScheduled", { date })}
      </span>
      <button
        type="button"
        onClick={() => cancel.mutate()}
        disabled={cancel.isPending}
        className="text-destructive shrink-0 rounded-full border border-[color:var(--destructive)]/40 px-3 py-1.5 text-[12px] font-semibold disabled:opacity-60"
      >
        {cancel.isPending ? t("cancelling") : t("bannerCancel")}
      </button>
    </div>
  )
}
