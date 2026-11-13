"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconCheck } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useLesson, useLessonTotals, usePath } from "../hooks"
import { hourOptions } from "../weekly-goal"
import { useSpent } from "./lesson-bar"

/**
 * The week's scale: the time the learner has studied in lessons since Monday
 * against the hours a week they said they can give, filled like the scale of
 * the level. It is the program's own goal, set when the program is built and
 * changed here at any time; without one the week's time is shown on its own and
 * the goal is asked for.
 */
export function WeeklyScale({
  language,
  nativeLanguage,
  goalMinutes,
}: {
  language: string
  nativeLanguage: string
  /** The goal in minutes; 0 when the learner gave none. */
  goalMinutes: number
}) {
  const t = useTranslations("path.week")
  const spent = useSpent()
  const { lesson } = useLesson()
  const totals = useLessonTotals(!!lesson)
  const { setGoal } = usePath(language, nativeLanguage)
  const [changing, setChanging] = useState(false)

  const week = totals.data?.week
  const goalHours = goalMinutes / 60
  const share = week !== undefined && goalMinutes > 0 ? Math.min(1, week / 60 / goalMinutes) : 0
  const reached = week !== undefined && goalMinutes > 0 && week / 60 >= goalMinutes
  const picking = changing || goalMinutes === 0

  return (
    <section
      aria-label={t("title")}
      className="bg-background border-border space-y-2.5 rounded-lg border p-4"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-sm font-semibold">{t("title")}</h2>
        <p className="text-muted-foreground text-xs tabular-nums">
          {week === undefined
            ? "–"
            : goalMinutes > 0
              ? t("done", { done: spent(week), goal: spent(goalMinutes * 60) })
              : spent(week)}
        </p>
      </div>

      {goalMinutes > 0 && (
        <div
          role="progressbar"
          aria-label={t("title")}
          aria-valuemin={0}
          aria-valuemax={goalMinutes}
          aria-valuenow={Math.min(goalMinutes, Math.round((week ?? 0) / 60))}
          aria-valuetext={
            week === undefined
              ? undefined
              : t("done", { done: spent(week), goal: spent(goalMinutes * 60) })
          }
          className="bg-border h-2 overflow-hidden rounded-full"
        >
          <div
            className="bg-success h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none"
            style={{ width: `${share * 100}%` }}
          />
        </div>
      )}

      {reached && (
        <p className="text-success flex items-center gap-1 text-xs font-medium">
          <IconCheck size={14} aria-hidden="true" />
          {t("reached")}
        </p>
      )}

      {picking ? (
        <div className="space-y-1.5">
          <p className="text-muted-foreground text-xs leading-relaxed">
            {t(goalMinutes === 0 ? "askGoal" : "pickGoal")}
          </p>
          <div role="group" aria-label={t("goal")} className="flex flex-wrap gap-2">
            {hourOptions.map((n) => (
              <Button
                key={n}
                type="button"
                size="sm"
                disabled={setGoal.isPending}
                variant={goalHours === n ? "default" : "outline"}
                aria-pressed={goalHours === n}
                onClick={() =>
                  setGoal.mutate(goalHours === n ? 0 : n, {
                    onSuccess: () => setChanging(false),
                  })
                }
                className="h-auto rounded-full px-3 py-1 text-xs"
              >
                {t("hoursValue", { n })}
              </Button>
            ))}
          </div>
        </div>
      ) : (
        <Button
          type="button"
          variant="ghost"
          onClick={() => setChanging(true)}
          className="text-muted-foreground h-auto px-0 py-0 text-xs underline underline-offset-4 hover:bg-transparent"
        >
          {t("change", { goal: spent(goalMinutes * 60) })}
        </Button>
      )}

      {setGoal.isError && (
        <p role="alert" className="text-destructive text-xs">
          {setGoal.error.message}
        </p>
      )}
    </section>
  )
}
