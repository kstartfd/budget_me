"use client"

import { useTranslations } from "next-intl"

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"

export function ItemPhotoViewer({
  src,
  title,
  onClose,
}: {
  src: string | null
  title: string
  onClose: () => void
}) {
  const t = useTranslations("lists")

  return (
    <Dialog open={src !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="bg-card flex max-h-[calc(100dvh-2rem)] max-w-[min(94vw,900px)] flex-col gap-3 overflow-hidden rounded-2xl p-3 sm:max-w-[min(94vw,900px)]">
        <DialogTitle className="truncate pr-10 text-sm font-semibold">
          {t("photoOf", { name: title })}
        </DialogTitle>
        {src && (
          // Same-origin upload; preserving the full image matters more than cropping.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={src}
            alt={t("photoOf", { name: title })}
            className="max-h-[calc(100dvh-7rem)] w-full rounded-xl bg-secondary object-contain"
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
