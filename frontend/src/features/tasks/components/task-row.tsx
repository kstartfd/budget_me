"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { ArrowRightIcon, CheckIcon, PencilIcon, TrashIcon } from "@/components/app-icons"
import { cn } from "@/lib/utils"

import type { Task } from "../schemas"

/**
 * One task row: a big round checkbox, the title (+ note), and the actions —
 * edit, push to tomorrow (open tasks only), delete.
 *
 * The actions are plain buttons rather than a swipe: this Mini App is used in
 * a browser as often as on a phone, and a mouse can neither swipe nor discover
 * that it should (the chip rows in globals.css carry the scar from exactly
 * that). The checkbox is the 44px target; the actions sit at the row's usual 30.
 *
 * Delete asks first — it swaps the icon row for an explicit Отмена/Удалить
 * pair rather than firing on the first tap, so a mistap in a scrolling list
 * never silently removes a task.
 */
export function TaskRow({
  task,
  carriedSince,
  onToggle,
  onEdit,
  onPostpone,
  onDelete,
}: {
  task: Task
  /** Set when the task was planned before the day being viewed. */
  carriedSince: string | null
  onToggle: () => void
  onEdit: () => void
  onPostpone: () => void
  onDelete: () => void
}) {
  const t = useTranslations("day")
  const locale = useLocale()
  const done = task.doneOn !== null
  const [confirmingDelete, setConfirmingDelete] = useState(false)

  return (
    <div className="flex items-center gap-1 px-2 py-0.5">
      <button
        type="button"
        role="checkbox"
        aria-checked={done}
        aria-label={task.title}
        onClick={onToggle}
        className="flex size-11 shrink-0 items-center justify-center"
      >
        <span
          className={cn(
            "flex size-[23px] items-center justify-center rounded-full border-2 transition-colors",
            done
              ? "border-success bg-success text-primary-foreground"
              : "border-faint text-transparent",
          )}
        >
          <CheckIcon size={13} />
        </span>
      </button>

      <div className="min-w-0 flex-1 py-2.5">
        <div
          className={cn(
            "truncate text-[14.5px] font-medium",
            done && "text-faint line-through",
          )}
        >
          {task.title}
        </div>
        <div className="mt-0.5 flex items-center gap-2">
          {task.note && (
            <span className="text-muted-foreground truncate text-[12px]">
              {task.note}
            </span>
          )}
          {carriedSince && !done && (
            <span className="text-warn border-warn/30 bg-warn/10 shrink-0 rounded-full border px-1.5 py-[1px] font-mono text-[10px] font-semibold">
              {t("since", { date: shortDate(carriedSince, locale) })}
            </span>
          )}
        </div>
      </div>

      {confirmingDelete ? (
        <div className="flex shrink-0 items-center gap-1.5 pr-1">
          <button
            type="button"
            onClick={() => setConfirmingDelete(false)}
            className="border-border bg-secondary text-foreground rounded-full border px-2.5 py-[7px] text-[11.5px] font-semibold"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            autoFocus
            onClick={onDelete}
            className="bg-destructive rounded-full px-2.5 py-[7px] text-[11.5px] font-semibold text-white"
          >
            {t("confirmDelete")}
          </button>
        </div>
      ) : (
        <div className="flex shrink-0 items-center gap-1.5 pr-1">
          <button
            type="button"
            aria-label={t("editAria", { name: task.title })}
            onClick={onEdit}
            className="border-border bg-secondary text-muted-foreground hover:text-foreground flex size-[30px] items-center justify-center rounded-full border transition-colors"
          >
            <PencilIcon />
          </button>
          {!done && (
            <button
              type="button"
              aria-label={t("postponeAria", { name: task.title })}
              title={t("postpone")}
              onClick={onPostpone}
              className="border-border bg-secondary text-muted-foreground hover:text-foreground flex size-[30px] items-center justify-center rounded-full border transition-colors"
            >
              <ArrowRightIcon />
            </button>
          )}
          <button
            type="button"
            aria-label={t("deleteAria", { name: task.title })}
            onClick={() => setConfirmingDelete(true)}
            className="border-border bg-secondary text-muted-foreground hover:text-destructive flex size-[30px] items-center justify-center rounded-full border transition-colors"
          >
            <TrashIcon />
          </button>
        </div>
      )}
    </div>
  )
}

/** "9 сент" / "Sep 9" — the day a carried-over task was originally planned for. */
function shortDate(iso: string, locale: string): string {
  const [y, m, d] = iso.split("-").map(Number)

  return new Intl.DateTimeFormat(locale, { day: "numeric", month: "short" })
    .format(new Date(y, m - 1, d))
    .replace(".", "")
}
