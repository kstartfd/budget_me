"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { shiftDate } from "@/lib/month"
import { cn } from "@/lib/utils"

import { useCreateTask } from "../mutations"

/**
 * Add a task. One required field and one optional one, so it stays a single
 * step — no numpad, no two-step flow like the expense sheet needs.
 *
 * The day defaults to the day being viewed (not to today): adding while looking
 * at tomorrow should plan for tomorrow.
 */
export function AddTaskSheet({
  date,
  today,
  onClose,
}: {
  date: string
  today: string
  onClose: () => void
}) {
  const t = useTranslations("day")
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const create = useCreateTask()

  const [title, setTitle] = useState("")
  const [note, setNote] = useState("")
  const [plannedFor, setPlannedFor] = useState(date)

  const days = [date, shiftDate(date, 1)]
  const canSubmit = title.trim().length > 0 && !create.isPending

  function submit() {
    if (!canSubmit) return

    create.mutate(
      { title: title.trim(), note: note.trim() || null, plannedFor },
      { onSuccess: onClose },
    )
  }

  function dayLabelFor(iso: string): string {
    if (iso === today) return tCommon("today")
    if (iso === shiftDate(today, 1)) return t("tomorrow")

    const [y, m, d] = iso.split("-").map(Number)

    return new Intl.DateTimeFormat(locale, { day: "numeric", month: "short" })
      .format(new Date(y, m - 1, d))
      .replace(".", "")
  }

  return (
    <BottomSheet title={t("addTitle")} onClose={onClose}>
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

        <div className="bg-secondary flex gap-1 rounded-2xl p-1">
          {days.map((iso) => {
            const on = iso === plannedFor
            return (
              <button
                key={iso}
                type="button"
                aria-pressed={on}
                onClick={() => setPlannedFor(iso)}
                className={cn(
                  "flex-1 rounded-[13px] py-[9px] text-[13.5px] font-semibold transition-colors",
                  on ? "bg-card text-foreground shadow-[var(--shadow-card)]" : "text-faint",
                )}
              >
                {dayLabelFor(iso)}
              </button>
            )
          })}
        </div>

        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          className="bg-primary text-primary-foreground mt-1 w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-40"
        >
          {t("addSubmit")}
        </button>
      </div>
    </BottomSheet>
  )
}
