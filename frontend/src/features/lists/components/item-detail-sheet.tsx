"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconPencil, IconTrash } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { useIsDesktop } from "@/hooks/use-is-desktop"

import { useDeleteItem } from "../mutations"
import type { Item } from "../schemas"
import { ItemPhotoViewer } from "./item-photo-viewer"
import { StarRating } from "./star-rating"
import { VerdictBadge } from "./verdict-badge"

export function ItemDetailSheet({
  item,
  listId,
  onClose,
  onEdit,
}: {
  item: Item
  listId: string
  onClose: () => void
  onEdit: () => void
}) {
  const t = useTranslations("lists")
  const isDesktop = useIsDesktop()
  const remove = useDeleteItem(listId)
  const [viewingPhoto, setViewingPhoto] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)

  function close() {
    if (viewingPhoto) setViewingPhoto(false)
    else onClose()
  }

  const content = (
    <div className="max-h-[calc(100dvh-8rem)] space-y-4 overflow-y-auto pt-4">
      <div>
        <h2 className="text-[20px] font-bold leading-tight break-words">
          {item.title}
        </h2>
        {item.purpose && (
          <p className="text-muted-foreground mt-1 text-sm break-words">
            {item.purpose}
          </p>
        )}
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <VerdictBadge verdict={item.verdict} />
          <div className="flex items-center gap-1.5">
            <StarRating rating={item.rating} size={19} />
            <span className="text-muted-foreground font-mono text-xs">
              {item.rating ? `${item.rating}/10` : "—/10"}
            </span>
          </div>
        </div>
      </div>

      {item.note && (
        <div className="bg-secondary rounded-2xl p-4">
          <p className="text-faint text-xs font-medium">{t("noteLabel")}</p>
          <p className="mt-1 text-sm leading-relaxed break-words whitespace-pre-wrap">
            {item.note}
          </p>
        </div>
      )}

      {item.photoUrl && (
        <div>
          <p className="text-faint mb-2 text-xs font-medium">
            {t("photoLabel")}
          </p>
          <button
            type="button"
            onClick={() => setViewingPhoto(true)}
            aria-label={t("viewPhotoOf", { name: item.title })}
            className="border-border bg-secondary focus-visible:ring-primary flex w-full flex-col overflow-hidden rounded-2xl border text-left outline-none focus-visible:ring-2"
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={item.photoUrl}
              alt=""
              className="h-48 w-full object-contain"
            />
            <span className="text-muted-foreground border-border w-full border-t px-3 py-2 text-xs font-medium">
              {t("viewPhoto")}
            </span>
          </button>
        </div>
      )}

      {confirmingDelete ? (
        <div className="border-destructive/30 bg-destructive/5 rounded-2xl border p-4">
          <p className="text-sm font-semibold">{t("deleteItemWarning")}</p>
          <div className="mt-3 flex gap-2">
            <Button
              type="button"
              variant="secondary"
              className="h-11 flex-1 rounded-xl text-sm"
              onClick={() => setConfirmingDelete(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              className="h-11 flex-1 rounded-xl text-sm"
              disabled={remove.isPending}
              onClick={() => remove.mutate(item.id, { onSuccess: onClose })}
            >
              {t("confirmDelete")}
            </Button>
          </div>
          {remove.isError && (
            <p role="alert" className="text-destructive mt-2 text-xs">
              {t("deleteItemError")}
            </p>
          )}
        </div>
      ) : (
        <div className="flex gap-2 pb-1">
          <Button
            type="button"
            variant="secondary"
            className="h-11 flex-1 gap-2 rounded-xl text-sm font-semibold"
            onClick={onEdit}
          >
            <IconPencil size={17} />
            {t("editItem")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            className="h-11 flex-1 gap-2 rounded-xl text-sm font-semibold"
            onClick={() => setConfirmingDelete(true)}
          >
            <IconTrash size={17} />
            {t("confirmDelete")}
          </Button>
        </div>
      )}
    </div>
  )

  return (
    <>
      {isDesktop ? (
        <Dialog open onOpenChange={(open) => !open && close()}>
          <DialogContent
            aria-describedby={undefined}
            className="bg-card border-border gap-0 rounded-[26px] p-6 sm:max-w-[480px]"
          >
            <DialogTitle className="text-lg font-bold">
              {t("itemDetails")}
            </DialogTitle>
            {content}
          </DialogContent>
        </Dialog>
      ) : (
        <BottomSheet title={t("itemDetails")} onClose={close}>
          {content}
        </BottomSheet>
      )}
      <ItemPhotoViewer
        src={viewingPhoto ? item.photoUrl : null}
        title={item.title}
        onClose={() => setViewingPhoto(false)}
      />
    </>
  )
}
