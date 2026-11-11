"use client"

import { useTranslations } from "next-intl"
import type { PathTopic } from "../schemas"
import { SkillIcon, skillBar, skillOrder } from "./skills"

/**
 * ProgramScale shows what a program is made of and how far along it is: the
 * whole programme as one line, and under it a line for every skill the program
 * teaches, so the learner sees at a glance that it has reading, listening and
 * speaking and not only exercises, and how much of each is done.
 */
export function ProgramScale({ topics }: { topics: PathTopic[] }) {
  const t = useTranslations("path")

  const total = topics.length
  const done = topics.filter((topic) => topic.status === "completed").length

  const rows = skillOrder
    .map((skill) => {
      const own = topics.filter((topic) => topic.skill === skill)

      return {
        skill,
        total: own.length,
        done: own.filter((topic) => topic.status === "completed").length,
      }
    })
    .filter((row) => row.total > 0)

  if (total === 0) return null

  return (
    <section
      aria-label={t("scaleProgram")}
      className="bg-background border-border space-y-3.5 rounded-lg border p-4"
    >
      <div className="space-y-1.5">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="text-sm font-semibold">{t("scaleProgram")}</h2>
          <p className="text-muted-foreground text-xs tabular-nums">
            {t("scaleDone", { done, total })}
          </p>
        </div>
        <div
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={done}
          className="bg-border h-2 overflow-hidden rounded-full"
        >
          <div
            className="bg-success h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none"
            style={{ width: `${(done / total) * 100}%` }}
          />
        </div>
      </div>

      <ul className="grid gap-x-5 gap-y-2.5 sm:grid-cols-2">
        {rows.map((row) => (
          <li key={row.skill} className="space-y-1">
            <div className="flex items-center gap-1.5 text-[13px]">
              <SkillIcon skill={row.skill} />
              <span className="min-w-0 flex-1 truncate">{t(`skill.${row.skill}`)}</span>
              <span className="text-muted-foreground text-xs tabular-nums">
                {row.done}/{row.total}
              </span>
            </div>
            <div
              role="progressbar"
              aria-label={t(`skill.${row.skill}`)}
              aria-valuemin={0}
              aria-valuemax={row.total}
              aria-valuenow={row.done}
              className="flex gap-0.5"
            >
              {Array.from({ length: row.total }, (_, i) => (
                <span
                  key={i}
                  className={
                    i < row.done
                      ? `h-1.5 flex-1 rounded-full ${skillBar(row.skill)}`
                      : "bg-border h-1.5 flex-1 rounded-full"
                  }
                />
              ))}
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
