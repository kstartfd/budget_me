"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { ChevronLeftIcon, ChevronRightIcon } from "@/components/app-icons"
import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { shiftDate } from "@/lib/month"

import { useCreateTask, useUpdateTask } from "../mutations"
import type { Task } from "../schemas"

/**
 * Add-or-edit a task. One required field and one optional one, so it stays a
 * single step — no numpad, no two-step flow like the expense sheet needs.
 *
 * The day picker is a stepper (‹ date ›), not a couple of quick-pick pills:
 * the day screen itself lets you browse arbitrarily far, so a task has to be
 * plannable that far out too — a carried-over task being edited may already
 * be several days in the past. Defaults to the day being viewed (not to
 * today): adding while looking at tomorrow should plan for tomorrow.
 */
export function AddTaskSheet({
  date,
  today,
  task,
  onClose,
}: {
  date: string
  today: string
  /** Present in edit mode — prefills the fields and PUTs instead of POSTing. */
  task?: Task
  onClose: () => void
}) {
  const t = useTranslations("day")
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const create = useCreateTask()
  const update = useUpdateTask()

  const [title, setTitle] = useState(task?.title ?? "")
  const [note, setNote] = useState(task?.note ?? "")
  const [plannedFor, setPlannedFor] = useState(task?.plannedFor ?? date)

  const pending = create.isPending || update.isPending
  const canSubmit = title.trim().length > 0 && !pending

  function submit() {
    if (!canSubmit) return

    const trimmedNote = note.trim() || null

    if (task) {
      update.mutate(
        { id: task.id, title: title.trim(), note: trimmedNote, plannedFor },
        { onSuccess: onClose },
      )

      return
    }

    create.mutate(
      { title: title.trim(), note: trimmedNote, plannedFor },
      { onSuccess: onClose },
    )
  }

  function dayLabelFor(iso: string): string {
    if (iso === today) return tCommon("today")
    if (iso === shiftDate(today, 1)) return t("tomorrow")

    const [y, m, d] = iso.split("-").map(Number)

    return new Intl.DateTimeFormat(locale, {
      day: "numeric",
      month: "long",
      weekday: "short",
    })
      .format(new Date(y, m - 1, d))
      .replace(/\.$/, "")
  }

  return (
    <BottomSheet title={task ? t("editTitle") : t("addTitle")} onClose={onClose}>
      <div className="flex flex-col gap-3 pt-4">
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
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("notePlaceholder")}
          maxLength={500}
          className="border-border bg-secondary text-foreground placeholder:text-faint w-full rounded-2xl border px-4 py-3 text-[14px] outline-none"
        />

        <div className="bg-secondary flex items-center gap-1 rounded-2xl p-1.5">
          <button
            type="button"
            aria-label={t("prevDay")}
            onClick={() => setPlannedFor(shiftDate(plannedFor, -1))}
            className="border-border bg-card text-foreground flex size-9 shrink-0 items-center justify-center rounded-xl border"
          >
            <ChevronLeftIcon size={18} />
          </button>
          <div className="min-w-0 flex-1 truncate text-center text-[13px] font-semibold">
            {dayLabelFor(plannedFor)}
          </div>
          <button
            type="button"
            aria-label={t("nextDay")}
            onClick={() => setPlannedFor(shiftDate(plannedFor, 1))}
            className="border-border bg-card text-foreground flex size-9 shrink-0 items-center justify-center rounded-xl border"
          >
            <ChevronRightIcon size={18} />
          </button>
        </div>
        {plannedFor !== today && (
          <button
            type="button"
            onClick={() => setPlannedFor(today)}
            className="text-primary -mt-1.5 self-center text-[12px] font-semibold"
          >
            {t("backToToday")}
          </button>
        )}

        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="bg-primary text-primary-foreground mt-1 w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-40"
        >
          {task ? t("editSubmit") : t("addSubmit")}
        </button>
      </div>
    </BottomSheet>
  )
}
