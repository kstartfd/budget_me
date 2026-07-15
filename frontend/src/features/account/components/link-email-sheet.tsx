"use client"

import { useTranslations } from "next-intl"

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { ApiError } from "@/features/auth"
import { BottomSheet, useBudgetUI } from "@/features/budget"
import { useIsDesktop } from "@/hooks/use-is-desktop"

import { useLinkEmail } from "../mutations"
import type { LinkEmailFormValues } from "../schemas"
import { LinkEmailForm } from "../forms/link-email-form"

/**
 * "Add email and password" surface: a bottom sheet on mobile, a centered dialog
 * on desktop, the same LinkEmailForm inside both. Owns the link mutation, the
 * success toast, and the 409 "already linked to another account" toast (the
 * sheet stays open on conflict so the visitor can fix the email). The identity
 * row itself flips to "linked" via the session-user patch in useLinkEmail.
 */
export function LinkEmailSheet({ onClose }: { onClose: () => void }) {
  const isDesktop = useIsDesktop()
  const t = useTranslations("account")
  const { showToast } = useBudgetUI()
  const link = useLinkEmail()

  function handleSubmit(values: LinkEmailFormValues) {
    link.mutate(values, {
      onSuccess: () => {
        onClose()
        showToast("success", t("emailLinked"))
      },
      onError: (err) => {
        showToast(
          "danger",
          err instanceof ApiError && err.status === 409
            ? t("emailConflict")
            : t("genericError"),
        )
      },
    })
  }

  const form = (
    <LinkEmailForm
      submitting={link.isPending}
      onSubmit={handleSubmit}
      onCancel={onClose}
    />
  )

  if (isDesktop) {
    return (
      <Dialog open onOpenChange={(open) => !open && onClose()}>
        <DialogContent
          aria-describedby={undefined}
          className="bg-card border-border gap-0 rounded-[26px] p-6 sm:max-w-[420px]"
        >
          <DialogTitle className="text-lg font-bold tracking-[-0.02em]">
            {t("emailTitle")}
          </DialogTitle>
          {form}
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <BottomSheet title={t("emailTitle")} onClose={onClose}>
      {form}
    </BottomSheet>
  )
}
