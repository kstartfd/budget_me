"use client"

import { useTranslations } from "next-intl"

/** StartError says why a test could not be opened: the AI agent writes it. */
export function StartError({ message }: { message: string }) {
  const t = useTranslations("path")

  return (
    <div role="alert" className="border-destructive/40 space-y-1 rounded-md border p-3 text-sm">
      <p className="text-destructive font-semibold">{t("startFailed")}</p>
      <p className="text-muted-foreground text-[13px] leading-relaxed">{t("startFailedHint")}</p>
      <p className="text-muted-foreground text-xs">{message}</p>
    </div>
  )
}
