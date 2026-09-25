"use client"

import { useEffect, useRef, useState, type ChangeEvent } from "react"
import { useTranslations } from "next-intl"
import { IconPhoto, IconPhotoPlus, IconTrash } from "@tabler/icons-react"

import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { cn } from "@/lib/utils"

import {
  useCreateItem,
  useDeleteItemPhoto,
  useUpdateItem,
  useUploadItemPhoto,
} from "../mutations"
import type { Item, Verdict } from "../schemas"
import { ItemPhotoViewer } from "./item-photo-viewer"
import { StarRating } from "./star-rating"

const verdicts: Verdict[] = ["unknown", "works", "meh", "no"]

/**
 * Add-or-edit an item. `item` present → edit (PUT), absent → create (POST)
 * into `listId` — same dual-purpose shape as tasks' AddTaskSheet.
 */
export function ItemSheet({
  listId,
  item,
  onClose,
}: {
  listId: string
  item?: Item
  onClose: () => void
}) {
  const t = useTranslations("lists")
  const create = useCreateItem(listId)
  const update = useUpdateItem(listId)
  const uploadPhoto = useUploadItemPhoto(listId)
  const removePhoto = useDeleteItemPhoto(listId)

  const [title, setTitle] = useState(item?.title ?? "")
  const [purpose, setPurpose] = useState(item?.purpose ?? "")
  const [verdict, setVerdict] = useState<Verdict>(item?.verdict ?? "unknown")
  const [rating, setRating] = useState<number | null>(item?.rating ?? null)
  const [note, setNote] = useState(item?.note ?? "")
  const [photoUrl, setPhotoUrl] = useState(item?.photoUrl ?? null)
  const [createdItem, setCreatedItem] = useState<Item | null>(null)
  const [selectedPhoto, setSelectedPhoto] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [viewingPhoto, setViewingPhoto] = useState(false)
  const [confirmingPhotoDelete, setConfirmingPhotoDelete] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  useEffect(
    () => () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    },
    [previewUrl],
  )
  const activeItem = createdItem ?? item
  const visiblePhoto = previewUrl ?? photoUrl

  const pending =
    create.isPending ||
    update.isPending ||
    uploadPhoto.isPending ||
    removePhoto.isPending
  const canSubmit = title.trim().length > 0 && !pending

  function submit() {
    if (!canSubmit) return

    const shared = {
      title: title.trim(),
      purpose: purpose.trim() || null,
      verdict,
      rating,
      note: note.trim() || null,
    }

    if (activeItem) {
      update.mutate(
        { id: activeItem.id, ...shared },
        {
          onSuccess: () => {
            if (selectedPhoto) {
              uploadPhoto.mutate(
                { id: activeItem.id, photo: selectedPhoto },
                { onSuccess: onClose },
              )
            } else {
              onClose()
            }
          },
        },
      )

      return
    }

    create.mutate(
      { listId, ...shared },
      {
        onSuccess: (created) => {
          setCreatedItem(created)
          if (!selectedPhoto) {
            onClose()
            return
          }
          uploadPhoto.mutate(
            { id: created.id, photo: selectedPhoto },
            {
              onSuccess: onClose,
            },
          )
        },
      },
    )
  }

  const photoPending = pending

  function handlePhotoChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]

    e.target.value = ""

    if (!file) return

    if (!activeItem) {
      setSelectedPhoto(file)
      setPreviewUrl(URL.createObjectURL(file))
      return
    }

    setSelectedPhoto(null)
    setPreviewUrl(null)
    uploadPhoto.mutate(
      { id: activeItem.id, photo: file },
      {
        onSuccess: (updated) => {
          setPhotoUrl(updated.photoUrl)
          setConfirmingPhotoDelete(false)
        },
      },
    )
  }

  return (
    <BottomSheet
      title={activeItem ? t("editItemTitle") : t("addItemTitle")}
      onClose={() => (viewingPhoto ? setViewingPhoto(false) : onClose())}
    >
      <div className="flex max-h-[calc(100dvh-7rem)] flex-col gap-3 overflow-y-auto pt-4">
        <input
          autoFocus
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("titlePlaceholder")}
          maxLength={200}
          className="border-primary bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3.5 text-[15px] outline-none"
        />
        <input
          value={purpose}
          onChange={(e) => setPurpose(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("purposePlaceholder")}
          maxLength={200}
          className="border-border bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3 text-[14px] outline-none"
        />
        <input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("notePlaceholder")}
          maxLength={500}
          className="border-border bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3 text-[14px] outline-none"
        />

        <div>
          <div className="text-faint mb-1.5 px-0.5 font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {t("photoLabel")}
          </div>
          {visiblePhoto ? (
            <button
              type="button"
              onClick={() => setViewingPhoto(true)}
              aria-label={t("viewPhoto")}
              className="border-border bg-secondary focus-visible:ring-primary flex w-full flex-col overflow-hidden rounded-2xl border outline-none focus-visible:ring-2"
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={visiblePhoto}
                alt=""
                className="h-40 w-full object-contain"
              />
              <span className="text-muted-foreground border-border w-full border-t px-3 py-2 text-left text-xs font-medium">
                {t("viewPhoto")}
              </span>
            </button>
          ) : (
            <div className="border-border bg-secondary text-muted-foreground flex h-36 flex-col items-center justify-center gap-2 rounded-2xl border border-dashed text-xs">
              <IconPhoto size={24} stroke={1.6} />
              {t("noPhoto")}
            </div>
          )}
          <input
            ref={fileInput}
            type="file"
            accept="image/*"
            onChange={handlePhotoChange}
            disabled={photoPending}
            className="sr-only"
            tabIndex={-1}
          />
          <div className="mt-2 flex items-center gap-2">
            <button
              type="button"
              onClick={() => fileInput.current?.click()}
              disabled={photoPending}
              className="border-border bg-secondary text-foreground focus-visible:ring-primary flex min-h-10 flex-1 items-center justify-center gap-2 rounded-xl border px-3 text-xs font-semibold outline-none focus-visible:ring-2 disabled:opacity-40"
            >
              <IconPhotoPlus size={17} stroke={1.8} />
              {visiblePhoto ? t("changePhoto") : t("addPhoto")}
            </button>
            {visiblePhoto && (
              <button
                type="button"
                onClick={() => setConfirmingPhotoDelete(true)}
                disabled={photoPending}
                className="border-border text-destructive focus-visible:ring-destructive flex min-h-10 items-center justify-center gap-2 rounded-xl border px-3 text-xs font-semibold outline-none focus-visible:ring-2 disabled:opacity-40"
              >
                <IconTrash size={16} stroke={1.8} />
                {t("removePhoto")}
              </button>
            )}
          </div>
          {selectedPhoto && (
            <p className="text-muted-foreground mt-1 truncate text-xs">
              {selectedPhoto.name}
            </p>
          )}
          {confirmingPhotoDelete && (
            <div className="border-border bg-secondary mt-2 rounded-xl border p-3">
              <p className="text-xs font-medium">{t("confirmRemovePhoto")}</p>
              <div className="mt-2 flex gap-2">
                <button
                  type="button"
                  onClick={() => setConfirmingPhotoDelete(false)}
                  className="border-border bg-card flex-1 rounded-lg border px-3 py-2 text-xs font-semibold"
                >
                  {t("cancel")}
                </button>
                <button
                  type="button"
                  disabled={photoPending}
                  onClick={() => {
                    if (selectedPhoto) {
                      setSelectedPhoto(null)
                      setPreviewUrl(null)
                      setConfirmingPhotoDelete(false)
                    } else if (activeItem) {
                      removePhoto.mutate(activeItem.id, {
                        onSuccess: () => {
                          setPhotoUrl(null)
                          setConfirmingPhotoDelete(false)
                        },
                      })
                    }
                  }}
                  className="bg-destructive flex-1 rounded-lg px-3 py-2 text-xs font-semibold text-white disabled:opacity-40"
                >
                  {t("confirmDelete")}
                </button>
              </div>
            </div>
          )}
          {(uploadPhoto.isError || removePhoto.isError) && (
            <p role="alert" className="text-destructive mt-2 text-xs">
              {t("photoError")}
            </p>
          )}
        </div>

        <div>
          <div className="text-faint mb-1.5 px-0.5 font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {t("verdictLabel")}
          </div>
          <div className="bg-secondary flex gap-1 rounded-2xl p-1">
            {verdicts.map((v) => {
              const on = v === verdict
              return (
                <button
                  key={v}
                  type="button"
                  aria-pressed={on}
                  onClick={() => setVerdict(v)}
                  className={cn(
                    "flex-1 rounded-[13px] py-[9px] text-[12.5px] font-semibold transition-colors",
                    on
                      ? "bg-card text-foreground shadow-[var(--shadow-card)]"
                      : "text-faint",
                  )}
                >
                  {t(`verdict.${v}`)}
                </button>
              )
            })}
          </div>
        </div>

        <div className="flex items-center justify-between px-0.5">
          <span className="text-faint font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
            {t("ratingLabel")}
          </span>
          <div className="flex items-center gap-2">
            <StarRating rating={rating} size={26} onChange={setRating} />
            <span className="text-muted-foreground w-[38px] font-mono text-[12px]">
              {rating ? `${rating}/10` : "—/10"}
            </span>
            {rating != null && (
              <button
                type="button"
                onClick={() => setRating(null)}
                className="text-faint text-[11px] font-semibold"
              >
                {t("clearRating")}
              </button>
            )}
          </div>
        </div>

        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="bg-primary text-primary-foreground mt-1 w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-40"
        >
          {activeItem ? t("editItemSubmit") : t("addItemSubmit")}
        </button>
      </div>
      <ItemPhotoViewer
        src={viewingPhoto ? visiblePhoto : null}
        title={title || t("photoLabel")}
        onClose={() => setViewingPhoto(false)}
      />
    </BottomSheet>
  )
}
