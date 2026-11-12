"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import Link from "next/link"
import { IconPencil, IconSettings, IconSparkles, IconTrash } from "@tabler/icons-react"
import { BottomBar } from "@/components/bottom-bar"
import { InlineMarkdown } from "@/components/inline-markdown"
import { Button } from "@/components/ui/button"
import { useAiSettings } from "@/features/ai"
import { EntryChat } from "@/features/chat"
import { cn } from "@/lib/utils"
import { useLearning } from "../hooks"
import type { Entry } from "../schemas"
import { EntryAudioButton } from "./audio-button"
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
  const { deleteEntry, saveEntry, fillEntry } = useLearning()
  const ai = useAiSettings()
  const [confirm, setConfirm] = useState(false)
  const page = variant === "page"
  return (
    <div className={cn(page ? "space-y-3.5" : "space-y-4")}>
      <div>
        {!page && (
          <div className="mb-2 flex items-center justify-between gap-2">
            <span className="bg-muted rounded-md px-2.5 py-[3px] text-xs font-semibold">
              {t(entry.kind)}
            </span>
            <Button
              variant="outline"
              size="icon"
              aria-label={t("editEntry")}
              onClick={onEdit}
              className="text-muted-foreground size-8"
            >
              <IconPencil size={13} />
            </Button>
          </div>
        )}
        <div className="flex items-start gap-3">
          <h2
            className={cn(
              "min-w-0 flex-1 leading-tight font-semibold tracking-[-0.01em] break-words",
              page ? "text-xl" : "text-2xl",
            )}
          >
            {entry.text}
          </h2>
          <EntryAudioButton entryId={entry.id} />
        </div>
        {entry.transcription && (
          <p className="text-muted-foreground mt-2 text-sm break-words">
            {entry.transcription}
          </p>
        )}
        {entry.translation && (
          <p
            className={cn(
              "text-foreground mt-2 leading-snug font-semibold break-words whitespace-pre-wrap",
              page ? "text-sm" : "text-[13px]",
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
            variant="outline"
            size="sm"
            disabled={saveEntry.isPending}
            aria-pressed={entry.status === status}
            className={cn(
              "h-7 px-3 text-xs font-medium",
              entry.status === status &&
                "bg-primary text-primary-foreground hover:bg-primary border-transparent hover:text-primary-foreground",
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
                  ? "border-border rounded-lg border p-3.5"
                  : "border-border border-t pt-3.5"
              }
            >
              <h3
                className={cn(
                  "text-faint mb-1.5 text-xs font-medium",
                )}
              >
                {t(field)}
              </h3>
              <p
                className={cn(
                  "text-foreground/90 break-words whitespace-pre-wrap",
                  page
                    ? "text-[13px] leading-[1.45]"
                    : "text-[13px] leading-[1.55]",
                )}
              >
                <InlineMarkdown text={entry[field]} />
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
          <div className="bg-muted rounded-lg p-4">
            <p className="text-muted-foreground text-sm leading-relaxed">
              {t("unfilledHint")}
            </p>
            <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2">
              {ai.isSuccess && !ai.data ? (
                // No agent yet: say so and lead to where it is connected.
                <Button asChild variant="outline" size="lg" className="text-[13px]">
                  <Link href="/learning/settings">
                    <IconSettings size={14} />
                    {t("fillNeedsAgent")}
                  </Link>
                </Button>
              ) : (
                <Button
                  size="lg"
                  disabled={fillEntry.isPending || ai.isPending}
                  onClick={() => fillEntry.mutate(entry.id)}
                  className="text-[13px]"
                >
                  <IconSparkles
                    size={14}
                    aria-hidden="true"
                    className={cn(fillEntry.isPending && "animate-pulse")}
                  />
                  {t(fillEntry.isPending ? "filling" : "fillEntry")}
                </Button>
              )}
              <Button
                variant="ghost"
                disabled={fillEntry.isPending}
                className="text-muted-foreground px-0 text-[13px] underline underline-offset-4"
                onClick={onEdit}
              >
                {t("fillEntryManual")}
              </Button>
            </div>
            {fillEntry.isError && (
              <p role="alert" className="text-destructive mt-2 text-xs leading-snug">
                {fillEntry.error.message}
              </p>
            )}
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
                  variant="outline"
                  size="lg"
                  disabled={deleteEntry.isPending}
                  onClick={() => setConfirm(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  variant="destructive"
                  size="lg"
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
              className="text-destructive hover:text-destructive px-0 text-xs"
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
            className="h-11 flex-1 text-sm"
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
            className="size-11"
            onClick={() => setConfirm(true)}
          >
            <IconTrash size={18} />
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
              variant="outline"
              className="h-9 flex-1"
              disabled={deleteEntry.isPending}
              onClick={() => setConfirm(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              variant="destructive"
              className="h-9 flex-1"
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
