"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconPencil, IconTrash } from "@tabler/icons-react"
import { BottomBar } from "@/components/bottom-bar"
import { Button } from "@/components/ui/button"
import { EntryChat } from "@/features/chat"
import { cn } from "@/lib/utils"
import { useLearning } from "../hooks"
import type { Entry } from "../schemas"
import { LearningModal } from "./learning-modal"

export function EntryDetail({
  entry,
  variant = "panel",
  onEdit,
  onDelete,
}: {
  entry: Entry
  variant?: "panel" | "page"
  onEdit: () => void
  onDelete: () => void
}) {
  const t = useTranslations("learning")
  const { deleteEntry, saveEntry } = useLearning()
  const [confirm, setConfirm] = useState(false)
  const page = variant === "page"
  return (
    <div className={cn(page ? "space-y-3.5" : "space-y-4")}>
      <div>
        {!page && (
          <div className="mb-2 flex items-center justify-between gap-2">
            <span className="text-faint text-xs">{t(entry.kind)}</span>
            <Button
              variant="ghost"
              size="icon"
              aria-label={t("editEntry")}
              onClick={onEdit}
              className="bg-secondary size-[30px] rounded-full"
            >
              <IconPencil size={15} />
            </Button>
          </div>
        )}
        <h2
          className={cn(
            "leading-[1.2] font-bold tracking-tight break-words",
            page ? "text-2xl" : "text-[26px]",
          )}
        >
          {entry.text}
        </h2>
        {entry.transcription && (
          <p className="text-muted-foreground mt-2 text-sm break-words">
            {entry.transcription}
          </p>
        )}
        {entry.translation && (
          <p
            className={cn(
              "text-primary mt-3 leading-[1.45] break-words whitespace-pre-wrap",
              page ? "text-[15px]" : "text-base",
            )}
          >
            {entry.translation}
          </p>
        )}
        {entry.source && (
          <p className="text-muted-foreground mt-3 text-xs break-words">
            {t("source")}: {entry.source}
          </p>
        )}
      </div>
      <div
        className="flex flex-wrap gap-2"
        role="group"
        aria-label={t("status")}
      >
        {(["inbox", "learning", "known"] as const).map((status) => (
          <Button
            key={status}
            variant="secondary"
            size="sm"
            disabled={saveEntry.isPending}
            aria-pressed={entry.status === status}
            className={cn(
              "h-auto rounded-full font-medium",
              page ? "px-4 py-2 text-[13px]" : "px-3 py-1.5 text-xs",
              entry.status === status
                ? "bg-primary/15 text-primary"
                : "text-foreground/75",
            )}
            onClick={() => saveEntry.mutate({ ...entry, status })}
          >
            {t(status)}
          </Button>
        ))}
      </div>
      {saveEntry.isError && (
        <p role="alert" className="text-destructive text-sm">
          {t("saveError")}
        </p>
      )}
      {(["explanation", "rule", "nativeUsage", "example"] as const).map(
        (field) =>
          entry[field] && (
            <section
              key={field}
              className={
                page
                  ? "bg-card border-border rounded-[14px] border p-3.5"
                  : "border-border border-t pt-3.5"
              }
            >
              <h3
                className={cn(
                  "text-faint mb-1.5",
                  page ? "text-[11px] tracking-[0.06em]" : "text-xs",
                )}
              >
                {t(field)}
              </h3>
              <p
                className={cn(
                  "text-foreground/90 break-words whitespace-pre-wrap",
                  page ? "text-[13px] leading-[1.45]" : "text-[13px] leading-[1.55]",
                )}
              >
                {entry[field]}
              </p>
            </section>
          ),
      )}
      {!entry.translation &&
        !entry.explanation &&
        !entry.nativeUsage &&
        !entry.example &&
        !entry.rule &&
        !entry.source && (
          <div className="bg-secondary rounded-2xl p-4">
            <p className="text-muted-foreground text-sm leading-relaxed">
              {t("unfilledHint")}
            </p>
            <Button
              variant="ghost"
              className="text-primary mt-2 px-0"
              onClick={onEdit}
            >
              {t("fillEntry")}
            </Button>
          </div>
        )}
      <EntryChat entryId={entry.id} page={page} />
      {!page && (
        <div className="border-border border-t pt-3.5">
          {confirm ? (
            <div className="space-y-3">
              <p className="text-sm">{t("deleteEntryWarning")}</p>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  disabled={deleteEntry.isPending}
                  onClick={() => setConfirm(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  variant="destructive"
                  disabled={deleteEntry.isPending}
                  onClick={() =>
                    deleteEntry.mutate(entry.id, { onSuccess: onDelete })
                  }
                >
                  {t("delete")}
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="ghost"
              className="text-destructive px-0 text-xs"
              onClick={() => setConfirm(true)}
            >
              <IconTrash size={15} />
              {t("deleteEntry")}
            </Button>
          )}
          {deleteEntry.isError && (
            <p role="alert" className="text-destructive mt-2 text-sm">
              {t("deleteError")}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

/** Phone entry screen footer: flip the entry between known and learning, or delete it. */
export function EntryActionBar({
  entry,
  onDelete,
}: {
  entry: Entry
  onDelete: () => void
}) {
  const t = useTranslations("learning")
  const { deleteEntry, saveEntry } = useLearning()
  const [confirm, setConfirm] = useState(false)
  const known = entry.status === "known"
  return (
    <>
      <BottomBar>
        {saveEntry.isError && (
          <p role="alert" className="text-destructive text-xs">
            {t("saveError")}
          </p>
        )}
        <div className="flex gap-2.5">
          <Button
            disabled={saveEntry.isPending}
            className="h-[50px] flex-1 rounded-[14px] text-[15px] font-bold"
            onClick={() =>
              saveEntry.mutate({
                ...entry,
                status: known ? "learning" : "known",
              })
            }
          >
            {t(known ? "markLearning" : "markKnown")}
          </Button>
          <Button
            variant="destructive"
            size="icon"
            aria-label={t("deleteEntry")}
            className="size-[50px] rounded-[14px]"
            onClick={() => setConfirm(true)}
          >
            <IconTrash size={20} />
          </Button>
        </div>
      </BottomBar>
      {confirm && (
        <LearningModal
          title={t("deleteEntry")}
          description={t("deleteEntryWarning")}
          onClose={() => setConfirm(false)}
        >
          <div className="flex gap-3">
            <Button
              variant="secondary"
              className="flex-1"
              disabled={deleteEntry.isPending}
              onClick={() => setConfirm(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              variant="destructive"
              className="flex-1"
              disabled={deleteEntry.isPending}
              onClick={() =>
                deleteEntry.mutate(entry.id, { onSuccess: onDelete })
              }
            >
              {t("delete")}
            </Button>
          </div>
          {deleteEntry.isError && (
            <p role="alert" className="text-destructive mt-3 text-sm">
              {t("deleteError")}
            </p>
          )}
        </LearningModal>
      )}
    </>
  )
}
