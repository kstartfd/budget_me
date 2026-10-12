"use client"

import { useFormatter, useTranslations } from "next-intl"

import { useAuth } from "@/features/auth"

import { useCancelAccountDeletion } from "../mutations"

/**
 * Shown on every screen while a deletion is pending — the account still works,
 * so without this the user would have no reminder that it is on a countdown.
 * Cancelling is one tap from wherever they are.
 */
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
      className="border-destructive/30 bg-[color:var(--destructive)]/10 mb-1 flex items-center gap-3 rounded-[16px] border px-3.5 py-3"
    >
      <span className="text-destructive min-w-0 flex-1 text-[12.5px] font-medium">
        {t("bannerScheduled", { date })}
      </span>
      <button
        type="button"
        onClick={() => cancel.mutate()}
        disabled={cancel.isPending}
        className="border-destructive/40 text-destructive shrink-0 rounded-full border px-3 py-1.5 text-[12px] font-semibold disabled:opacity-60"
      >
        {cancel.isPending ? t("cancelling") : t("bannerCancel")}
      </button>
    </div>
  )
}
