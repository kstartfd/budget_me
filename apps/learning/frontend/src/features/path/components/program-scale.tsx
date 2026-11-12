"use client"

import { useTranslations } from "next-intl"
import { awaitsExercises, topicProgress } from "../progress"
import type { PathTopic } from "../schemas"
import { SkillIcon, skillBar, skillOrder } from "./skills"

/**
 * ProgramScale shows what a program is made of and how far along it is: the
 * whole programme as one line, and under it a line for every skill the program
 * teaches, so the learner sees at a glance that it has reading, listening and
 * speaking and not only exercises, and how much of each is done.
 *
 * Progress counts by exercises: each topic is a segment that fills with the
 * share of its exercises answered right, a topic without a set yet is empty,
 * and a completed one is whole. A learner who asks for a new set sees that
 * topic's segment start over, with nothing else to recompute.
 */
export function ProgramScale({
  topics,
  preparing = false,
}: {
  topics: PathTopic[]
  /** The exercises are still being written: say so, and count what is there. */
  preparing?: boolean
}) {
  const t = useTranslations("path")

  const total = topics.length
  const done = topics.filter((topic) => topic.status === "completed").length
  const progress = topics.reduce((sum, topic) => sum + topicProgress(topic), 0)

  const exercises = topics.reduce(
    (sum, topic) => ({
      total: sum.total + topic.exercises.total,
      done: sum.done + topic.exercises.done,
    }),
    { total: 0, done: 0 },
  )
  const waiting = topics.filter(awaitsExercises).length

  const rows = skillOrder
    .map((skill) => {
      const own = topics.filter((topic) => topic.skill === skill)

      return {
        skill,
        topics: own,
        done: own.filter((topic) => topic.status === "completed").length,
      }
    })
    .filter((row) => row.topics.length > 0)

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
          aria-valuenow={progress}
          aria-valuetext={t("scaleDone", { done, total })}
          className="bg-border h-2 overflow-hidden rounded-full"
        >
          <div
            className="bg-success h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none"
            style={{ width: `${(progress / total) * 100}%` }}
          />
        </div>
        {(exercises.total > 0 || preparing) && (
          <p
            aria-live="polite"
            className="text-muted-foreground flex flex-wrap gap-x-2 text-xs tabular-nums"
          >
            {exercises.total > 0 && (
              <span>{t("scaleExercises", exercises)}</span>
            )}
            {preparing && waiting > 0 && (
              <span>
                {t("scalePreparing", { ready: total - waiting, total })}
              </span>
            )}
          </p>
        )}
      </div>

      <ul className="grid gap-x-5 gap-y-2.5 sm:grid-cols-2">
        {rows.map((row) => (
          <li key={row.skill} className="space-y-1">
            <div className="flex items-center gap-1.5 text-[13px]">
              <SkillIcon skill={row.skill} />
              <span className="min-w-0 flex-1 truncate">{t(`skill.${row.skill}`)}</span>
              <span className="text-muted-foreground text-xs tabular-nums">
                {row.done}/{row.topics.length}
              </span>
            </div>
            <div
              role="progressbar"
              aria-label={t(`skill.${row.skill}`)}
              aria-valuemin={0}
              aria-valuemax={row.topics.length}
              aria-valuenow={row.done}
              className="flex gap-0.5"
            >
              {row.topics.map((topic) => (
                <span
                  key={topic.id}
                  className="bg-border relative h-1.5 flex-1 overflow-hidden rounded-full"
                >
                  <span
                    className={`absolute inset-y-0 left-0 rounded-full transition-[width] duration-300 motion-reduce:transition-none ${skillBar(row.skill)}`}
                    style={{ width: `${topicProgress(topic) * 100}%` }}
                  />
                </span>
              ))}
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
