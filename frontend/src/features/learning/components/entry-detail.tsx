"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconPencil, IconTrash } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useLearning } from "../hooks"
import type { Entry } from "../schemas"

export function EntryDetail({
  entry,
  onEdit,
  onDelete,
}: {
  entry: Entry
  onEdit: () => void
  onDelete: () => void
}) {
  const t = useTranslations("learning")
  const { deleteEntry, saveEntry } = useLearning()
  const [confirm, setConfirm] = useState(false)
  return (
    <div className="space-y-5">
      <div>
        <div className="mb-2 flex items-center justify-between gap-2">
          <span className="text-faint text-xs">{t(entry.kind)}</span>
          <Button
            variant="ghost"
            size="icon"
            aria-label={t("editEntry")}
            onClick={onEdit}
            className="bg-secondary size-9 rounded-full"
          >
            <IconPencil size={16} />
          </Button>
        </div>
        <h2 className="text-[25px] leading-tight font-bold tracking-tight break-words">
          {entry.text}
        </h2>
        {entry.transcription && (
          <p className="text-muted-foreground mt-2 text-sm break-words">
            {entry.transcription}
          </p>
        )}
        {entry.translation && (
          <p className="text-primary mt-3 text-base leading-relaxed break-words whitespace-pre-wrap">
            {entry.translation}
          </p>
        )}
        {entry.source && (
          <p className="text-muted-foreground mt-3 text-sm break-words">
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
            className={`rounded-full text-xs ${entry.status === status ? "bg-primary/15 text-primary" : ""}`}
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
      {(["explanation", "nativeUsage", "example", "rule"] as const).map(
        (field) =>
          entry[field] && (
            <section key={field} className="border-border border-t pt-4">
              <h3 className="text-muted-foreground mb-2 text-xs font-medium">
                {t(field)}
              </h3>
              <p className="text-sm leading-relaxed break-words whitespace-pre-wrap">
                {entry[field]}
              </p>
            </section>
          ),
      )}
      {!entry.translation &&
        !entry.explanation &&
        !entry.nativeUsage &&
        !entry.example &&
        !entry.rule && (
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
      <div className="border-border border-t pt-4">
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
    </div>
  )
}
