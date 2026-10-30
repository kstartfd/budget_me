"use client"

import { useState } from "react"
import { useFormatter, useTranslations } from "next-intl"
import { IconChevronRight, IconTrash } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { LearningModal } from "@/features/learning/components/learning-modal"

import { useDeleteDialog, useDialog, useDialogs } from "../hooks"
import { Bubble } from "./chat-ui"

/**
 * The dialogs cleared from a topic's chat. They are kept for reading, and can be
 * deleted one by one; the chat itself always continues in a new dialog.
 */
export function DialogHistory({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("chat")
  const format = useFormatter()
  const dialogs = useDialogs(topicId)
  const [openId, setOpenId] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const dialog = useDialog(openId)
  const remove = useDeleteDialog(topicId)

  const when = (iso: string) =>
    format.dateTime(new Date(iso), {
      day: "numeric",
      month: "short",
      hour: "numeric",
      minute: "numeric",
    })

  if (openId) {
    return (
      <LearningModal
        title={dialog.data ? when(dialog.data.startedAt) : t("dialog")}
        onClose={onClose}
        onBack={() => {
          setOpenId(null)
          setConfirmDelete(false)
        }}
      >
        {dialog.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("historyLoadFailed")}
          </p>
        ) : !dialog.data ? (
          <Skeleton className="h-32 w-full" />
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              {dialog.data.messages.map((m) => (
                <Bubble key={m.id} role={m.role}>
                  {m.content}
                </Bubble>
              ))}
            </div>
            {confirmDelete ? (
              <div className="flex items-center gap-2">
                <span className="text-sm">{t("deleteDialogConfirm")}</span>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={remove.isPending}
                  onClick={() => setConfirmDelete(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={remove.isPending}
                  onClick={() =>
                    remove.mutate(openId, {
                      onSuccess: () => {
                        setOpenId(null)
                        setConfirmDelete(false)
                      },
                    })
                  }
                >
                  {t("deleteDialog")}
                </Button>
              </div>
            ) : (
              <Button
                variant="ghost"
                className="text-muted-foreground h-auto px-0 py-0 text-xs"
                onClick={() => setConfirmDelete(true)}
              >
                <IconTrash size={14} />
                {t("deleteDialog")}
              </Button>
            )}
            {remove.isError && (
              <p role="alert" className="text-destructive text-sm">
                {t("deleteDialogFailed")}
              </p>
            )}
          </div>
        )}
      </LearningModal>
    )
  }

  return (
    <LearningModal
      title={t("history")}
      description={t("historyHint")}
      onClose={onClose}
    >
      {dialogs.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("historyLoadFailed")}
        </p>
      ) : dialogs.isPending ? (
        <Skeleton className="h-24 w-full" />
      ) : dialogs.data.length === 0 ? (
        <p className="text-muted-foreground text-sm leading-relaxed">
          {t("historyEmpty")}
        </p>
      ) : (
        <ul className="space-y-2">
          {dialogs.data.map((d) => (
            <li key={d.id}>
              <button
                type="button"
                onClick={() => setOpenId(d.id)}
                className="bg-secondary focus-visible:ring-primary flex w-full items-center gap-3 rounded-2xl px-3.5 py-3 text-left outline-none focus-visible:ring-2"
              >
                <span className="min-w-0 flex-1">
                  <span className="line-clamp-2 text-sm leading-snug font-medium break-words">
                    {d.title || t("dialog")}
                  </span>
                  <span className="text-faint mt-1 block text-xs">
                    {when(d.startedAt)} ·{" "}
                    {t("count", { count: d.messageCount })}
                  </span>
                </span>
                <IconChevronRight size={16} className="text-faint shrink-0" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </LearningModal>
  )
}
