"use client"

import { useTranslations } from "next-intl"

import { ApiError } from "@/features/auth"
import { LearningModal } from "@/features/learning/components/learning-modal"

import { LinkEmailForm } from "../forms/link-email-form"
import { useLinkEmail } from "../mutations"
import type { LinkEmailFormValues } from "../schemas"

/**
 * The sheet stays open on failure so the email can be fixed; on success it
 * closes and the identity row flips to "linked" via the session-user patch.
 */
export function LinkEmailSheet({ onClose }: { onClose: () => void }) {
  const t = useTranslations("account")
  const link = useLinkEmail()

  function handleSubmit(values: LinkEmailFormValues) {
    link.mutate(values, { onSuccess: onClose })
  }

  const error = link.isError
    ? link.error instanceof ApiError && link.error.status === 409
      ? t("emailConflict")
      : t("genericError")
    : null

  return (
    <LearningModal
      title={t("emailTitle")}
      description={t("emailSubtitle")}
      onClose={onClose}
    >
      <LinkEmailForm
        submitting={link.isPending}
        error={error}
        onSubmit={handleSubmit}
        onCancel={onClose}
      />
    </LearningModal>
  )
}
