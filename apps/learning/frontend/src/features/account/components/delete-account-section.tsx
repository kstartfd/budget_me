"use client"

import { useState } from "react"
import { useFormatter, useTranslations } from "next-intl"

import { useAuth } from "@/features/auth"

import {
  useCancelAccountDeletion,
  useRequestAccountDeletion,
} from "../mutations"

/**
 * The danger zone. Deleting is deferred, not immediate: the request sets a
 * deadline and the account keeps working until it passes, so the destructive
 * step is a confirmation, not a point of no return. Once a request is in, the
 * section flips to "scheduled + cancel".
 */
export function DeleteAccountSection() {
  const t = useTranslations("account")
  const format = useFormatter()
  const { user } = useAuth()
  const [confirming, setConfirming] = useState(false)
  const request = useRequestAccountDeletion()
  const cancel = useCancelAccountDeletion()

  const deleteAt = user?.delete_at ?? null
  const deleteAtLabel = deleteAt
    ? format.dateTime(new Date(deleteAt), {
        day: "numeric",
        month: "long",
        year: "numeric",
      })
    : ""
  const failed = request.isError || cancel.isError

  return (
    <section className="flex flex-col gap-[9px]">
      <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
        {t("dangerKicker")}
      </span>

      <div className="bg-card rounded-[18px] border border-[color:var(--destructive)]/30 p-[14px] shadow-[var(--shadow-card)]">
        {deleteAt ? (
          <>
            <div className="text-destructive text-[14px] font-semibold">
              {t("deleteScheduled", { date: deleteAtLabel })}
            </div>
            <p className="text-muted-foreground mt-[3px] text-[11.5px]">
              {t("deleteScheduledHint")}
            </p>
            <button
              type="button"
              onClick={() => cancel.mutate()}
              disabled={cancel.isPending}
              className="border-border bg-secondary text-foreground mt-3 w-full rounded-xl border p-3 text-[13.5px] font-semibold disabled:opacity-60"
            >
              {cancel.isPending ? t("cancelling") : t("deleteCancel")}
            </button>
          </>
        ) : confirming ? (
          <>
            <div className="text-destructive text-[14px] font-semibold">
              {t("deleteConfirmTitle")}
            </div>
            <p className="text-muted-foreground mt-[3px] text-[11.5px]">
              {t("deleteConfirmHint")}
            </p>
            <div className="mt-3 flex gap-2">
              <button
                type="button"
                onClick={() => {
                  setConfirming(false)
                  request.reset()
                }}
                className="border-border bg-secondary text-foreground flex-1 rounded-xl border p-3 text-[13.5px] font-semibold"
              >
                {t("deleteKeep")}
              </button>
              <button
                type="button"
                onClick={() =>
                  request.mutate(undefined, {
                    onSuccess: () => setConfirming(false),
                  })
                }
                disabled={request.isPending}
                className="bg-destructive text-primary-foreground flex-1 rounded-xl p-3 text-[13.5px] font-bold disabled:opacity-60"
              >
                {request.isPending ? t("deleting") : t("deleteConfirm")}
              </button>
            </div>
          </>
        ) : (
          <div className="flex items-center gap-3">
            <div className="min-w-0 flex-1">
              <div className="text-[14px] font-semibold">{t("deleteTitle")}</div>
              <p className="text-muted-foreground mt-[3px] text-[11.5px]">
                {t("deleteHint")}
              </p>
            </div>
            <button
              type="button"
              onClick={() => setConfirming(true)}
              className="text-destructive shrink-0 rounded-xl border border-[color:var(--destructive)]/40 px-3.5 py-2 text-[12.5px] font-semibold"
            >
              {t("deleteAction")}
            </button>
          </div>
        )}

        {failed && (
          <p role="alert" className="text-destructive mt-3 text-xs">
            {t("deleteFailed")}
          </p>
        )}
      </div>
    </section>
  )
}
