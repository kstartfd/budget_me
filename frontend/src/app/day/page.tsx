"use client"

import { useCallback, useMemo, useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { useProvideAddAction } from "@/components/add-action"
import {
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@/components/app-icons"
import {
  AddTaskSheet,
  TaskRow,
  useDayTasks,
  useDeleteTask,
  usePostponeTask,
  useToggleTask,
  type Task,
} from "@/features/tasks"
import { useToday } from "@/features/settings"
import { dateTitle, shiftDate } from "@/lib/month"
import { cn } from "@/lib/utils"

import { DaySkeleton } from "./skeleton"

const card =
  "bg-card border-border rounded-[22px] border shadow-[var(--shadow-card)]"

/**
 * `/day` — the daily checklist (design PHONE «День» / WEB FRAME «День»).
 *
 * Nothing is ever moved between days: the list for a date is every task
 * planned on or before it that is still open, plus what was ticked on that
 * date (see features/tasks/api.ts). So an untouched task simply shows up again
 * tomorrow, wearing the day it was originally planned for.
 */
export default function DayPage() {
  const t = useTranslations("day")
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const today = useToday()

  const [date, setDate] = useState(today)
  // Add and edit share one sheet — null closed, "add" for a new task, a Task
  // to edit that one (prefilled).
  const [sheet, setSheet] = useState<Task | "add" | null>(null)
  const [showDone, setShowDone] = useState(false)

  // The nav's center ＋ belongs to this screen while it is open.
  useProvideAddAction(useCallback(() => setSheet("add"), []))

  const tasks = useDayTasks(date)
  const toggle = useToggleTask(date)
  const postpone = usePostponeTask()
  const remove = useDeleteTask()

  const { open, done } = useMemo(() => {
    const rows = tasks.data ?? []

    return {
      open: rows.filter((task) => task.doneOn === null),
      done: rows.filter((task) => task.doneOn !== null),
    }
  }, [tasks.data])

  if (tasks.isPending) return <DaySkeleton />

  if (tasks.isError) {
    return (
      <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
        {tCommon("loadError")}
      </p>
    )
  }

  const total = open.length + done.length
  const isToday = date === today

  return (
    <div className="anim-fade flex flex-col gap-4 lg:max-w-2xl">
      {/* date + progress */}
      <section className={`${card} px-5 py-[18px]`}>
        <div className="flex items-center gap-2">
          <button
            type="button"
            aria-label={t("prevDay")}
            onClick={() => setDate(shiftDate(date, -1))}
            className="border-border bg-secondary text-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px] border"
          >
            <ChevronLeftIcon />
          </button>
          <div className="min-w-0 flex-1 text-center">
            <div className="text-faint font-mono text-[10.5px] font-semibold tracking-[0.09em] uppercase">
              {isToday ? tCommon("today") : t("kicker")}
            </div>
            <div className="truncate text-[17px] font-bold tracking-[-0.02em]">
              {dateTitle(date, locale)}
            </div>
          </div>
          <button
            type="button"
            aria-label={t("nextDay")}
            onClick={() => setDate(shiftDate(date, 1))}
            className="border-border bg-secondary text-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px] border"
          >
            <ChevronRightIcon />
          </button>
        </div>

        <div className="mt-4 flex items-center gap-3">
          <div className="bg-secondary h-1.5 flex-1 overflow-hidden rounded-full">
            <div
              className="bg-success h-full rounded-full transition-[width]"
              style={{ width: `${total === 0 ? 0 : (done.length / total) * 100}%` }}
            />
          </div>
          <span className="text-muted-foreground shrink-0 font-mono text-[12px]">
            {t("progress", { done: done.length, total })}
          </span>
        </div>

        {!isToday && (
          <button
            type="button"
            onClick={() => setDate(today)}
            className="text-primary mt-3 text-[12.5px] font-semibold"
          >
            {t("backToToday")}
          </button>
        )}
      </section>

      {/* open tasks */}
      {open.length === 0 ? (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {total === 0 ? t("empty") : t("allDone")}
        </p>
      ) : (
        <div className={`${card} divide-rowline divide-y overflow-hidden`}>
          {open.map((task) => (
            <TaskRow
              key={task.id}
              task={task}
              carriedSince={task.plannedFor < date ? task.plannedFor : null}
              onToggle={() => toggle.mutate({ id: task.id, done: true })}
              onEdit={() => setSheet(task)}
              onPostpone={() =>
                postpone.mutate({
                  id: task.id,
                  title: task.title,
                  note: task.note,
                  to: shiftDate(date, 1),
                })
              }
              onDelete={() => remove.mutate(task.id)}
            />
          ))}
        </div>
      )}

      {/* Done work is collapsed: it is a tally, not a list you act on, and on a
          phone it would otherwise push the open tasks off the screen. */}
      {done.length > 0 && (
        <div className={`${card} overflow-hidden`}>
          <button
            type="button"
            aria-expanded={showDone}
            onClick={() => setShowDone((v) => !v)}
            className="flex w-full items-center justify-between px-[15px] py-[13px]"
          >
            <span className="text-muted-foreground text-[13px] font-semibold">
              {t("doneSection", { count: done.length })}
            </span>
            <span
              className={cn(
                "text-faint transition-transform",
                showDone && "rotate-180",
              )}
            >
              <ChevronDownIcon />
            </span>
          </button>
          {showDone && (
            <div className="divide-rowline border-rowline divide-y border-t">
              {done.map((task) => (
                <TaskRow
                  key={task.id}
                  task={task}
                  carriedSince={null}
                  onToggle={() => toggle.mutate({ id: task.id, done: false })}
                  onEdit={() => setSheet(task)}
                  onPostpone={() => undefined}
                  onDelete={() => remove.mutate(task.id)}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {sheet && (
        <AddTaskSheet
          date={date}
          today={today}
          task={sheet === "add" ? undefined : sheet}
          onClose={() => setSheet(null)}
        />
      )}
    </div>
  )
}
