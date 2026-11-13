"use client"

import { useEffect, useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronRight, IconChevronUp } from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import Link from "next/link"
import { useLesson, useRules } from "../hooks"
import { exercisesPreparing } from "../progress"
import {
  flashAttribute,
  flashMs,
  forgetReturnPoint,
  peekReturnPoint,
  rememberReturnPoint,
  rulesButtonAnchor,
} from "../return-point"
import type { Path } from "../schemas"
import { LessonBar, lessonBarHeight } from "./lesson-bar"
import { Mixed, rulesPath } from "./rule-card"
import { ProgramScale } from "./program-scale"
import { SkillIcon } from "./skills"
import { StartError } from "./start-error"
import { TopicRunner } from "./topic-runner"

const statusBadge: Record<
  string,
  { variant: "default" | "outline"; className?: string }
> = {
  not_started: { variant: "outline" },
  in_progress: { variant: "default" },
  completed: {
    variant: "outline",
    className: "bg-success/10 text-success border-transparent",
  },
}

const badgeShape = "h-auto rounded-full px-2.5 py-[3px] text-xs"

/** ProgramView is the active program: level, summary, topics, actions. */
export function ProgramView({
  path,
  onStartFinal,
  onReset,
  startError,
  starting,
  asOf,
}: {
  path: Path
  /** When the program was last read (ms): the moment its exercises are judged at. */
  asOf: number
  onStartFinal: () => void
  onReset: () => void
  startError: string | null
  starting: boolean
}) {
  const t = useTranslations("path")
  // Coming back from the rule book: the topic that was open opens again and the
  // link that was pressed is scrolled to and framed. The program is only drawn
  // once it is loaded, in the browser, so the way back can be read right here.
  const [left] = useState(() => {
    const point = peekReturnPoint()

    return point &&
      point.language === path.language &&
      point.translationLanguage === path.translationLanguage
      ? point
      : null
  })
  const [openTopic, setOpenTopic] = useState<string | null>(
    left?.topicId && path.topics.some((topic) => topic.id === left.topicId)
      ? left.topicId
      : null,
  )
  const [returnTo, setReturnTo] = useState<string | null>(left?.anchor ?? null)
  const [confirmReset, setConfirmReset] = useState(false)
  useEffect(() => {
    if (left) forgetReturnPoint()
  }, [left])
  useEffect(() => {
    if (!returnTo) return

    // The topic's rule arrives with its own request, so the link is waited for.
    let frame: number | undefined
    let unframe: number | undefined
    const giveUpAt = Date.now() + 6000
    const find = () => {
      const target = document.querySelector<HTMLElement>(
        `[data-return-anchor="${returnTo}"]`,
      )
      if (!target) {
        if (Date.now() < giveUpAt) frame = window.setTimeout(find, 120)
        else setReturnTo(null)

        return
      }

      target.scrollIntoView({ block: "center" })
      target.setAttribute(flashAttribute, "")
      unframe = window.setTimeout(() => {
        target.removeAttribute(flashAttribute)
        setReturnTo(null)
      }, flashMs)
    }
    find()

    return () => {
      window.clearTimeout(frame)
      window.clearTimeout(unframe)
      document
        .querySelectorAll(`[${flashAttribute}]`)
        .forEach((element) => element.removeAttribute(flashAttribute))
    }
  }, [returnTo])
  const done = path.topics.filter((topic) => topic.status === "completed").length
  // The rule book is there for a teacher whose language it covers; for any other
  // the request is refused and the button stays out.
  const rules = useRules(path.language)
  const finished = path.topics.length > 0 && done === path.topics.length
  // While a lesson runs its bar sticks under the app bar, and the open topic's
  // row sticks right under the bar.
  const { lesson } = useLesson()
  const lessonRunning = !!lesson
  // The server is still writing the program's exercises, so the scale says so.
  const preparing = exercisesPreparing(path, asOf)

  return (
    <div className="space-y-4">
      <section className="bg-background border-border space-y-3 rounded-lg border p-5">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-base font-semibold">
            {t("programTitle", { level: path.level })}
          </h2>
          <Badge variant="secondary" className={cn(badgeShape, "font-semibold")}>
            {t("progressTopics", { done, total: path.topics.length })}
          </Badge>
        </div>
        {path.summary && (
          <p className="text-muted-foreground text-sm leading-relaxed">
            <Mixed text={path.summary} />
          </p>
        )}
        {finished && (
          <div className="border-border bg-muted/40 space-y-1 rounded-md border p-3.5">
            <p className="text-sm font-semibold">{t("retakeTitle")}</p>
            <p className="text-muted-foreground text-[13px] leading-relaxed">
              {t("retakeHint")}
            </p>
          </div>
        )}
        {startError && <StartError message={startError} />}
        <div className="flex flex-wrap gap-2">
          <Button
            size="lg"
            variant={finished ? "default" : "secondary"}
            onClick={onStartFinal}
            disabled={starting}
            className="px-4 text-[13px]"
          >
            {starting ? t("starting") : t(finished ? "retakeButton" : "finalTest")}
          </Button>
          {rules.isSuccess && rules.data.length > 0 && (
            <Button asChild variant="outline" size="lg" className="px-4 text-[13px]">
              <Link
                href={rulesPath(path.language)}
                data-return-anchor={rulesButtonAnchor}
                onClick={() =>
                  rememberReturnPoint({
                    language: path.language,
                    translationLanguage: path.translationLanguage,
                    topicId: openTopic,
                    anchor: rulesButtonAnchor,
                  })
                }
              >
                {t("rulesOpen")}
              </Link>
            </Button>
          )}
          {confirmReset ? (
            <>
              <Button
                variant="destructive"
                size="lg"
                onClick={() => {
                  setConfirmReset(false)
                  onReset()
                }}
                className="px-4 text-[13px]"
              >
                {t("resetConfirm")}
              </Button>
              <Button
                variant="ghost"
                size="lg"
                onClick={() => setConfirmReset(false)}
                className="px-4 text-[13px]"
              >
                {t("cancel")}
              </Button>
            </>
          ) : (
            <Button
              variant="outline"
              size="lg"
              onClick={() => setConfirmReset(true)}
              className="px-4 text-[13px]"
            >
              {t("reset")}
            </Button>
          )}
        </div>
      </section>

      <LessonBar />

      <ProgramScale topics={path.topics} preparing={preparing} />

      <section className="space-y-3">
        {path.topics.map((topic, index) => {
          const open = openTopic === topic.id
          const newUnit =
            !!topic.unit && topic.unit !== path.topics[index - 1]?.unit

          return (
            <div key={topic.id} className="space-y-3">
              {newUnit && (
                <h2 className="text-muted-foreground pt-2 text-xs font-semibold tracking-wider uppercase">
                  {topic.unit}
                </h2>
              )}
            <div
              className={cn(
                // clip, not hidden: the open topic's row sticks under the app bar (61px tall, see
                // app-shell; none at lg) while its exercises scroll
                "bg-background min-w-0 overflow-clip rounded-lg border",
                topic.status === "in_progress"
                  ? "border-foreground"
                  : "border-border",
              )}
            >
              <button
                type="button"
                aria-expanded={open}
                onClick={() => setOpenTopic(open ? null : topic.id)}
                className={cn(
                  "hover:bg-muted/50 relative flex w-full min-w-0 cursor-pointer items-center gap-3 px-4 py-3.5 text-left transition-colors",
                  topic.status === "in_progress" && "bg-muted",
                  open &&
                    cn(
                      "border-border bg-background hover:bg-muted sticky z-10 border-b",
                      // the app bar is 61px, the lesson bar 52px (lessonBarHeight)
                      lessonRunning && lessonBarHeight === 52
                        ? "top-[113px] lg:top-[52px]"
                        : "top-[61px] lg:top-0",
                    ),
                  open && topic.status === "in_progress" && "bg-muted",
                )}
              >
                <span className="bg-secondary flex size-8 shrink-0 items-center justify-center rounded-full text-[13px] font-semibold">
                  {topic.position + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <h3 className="line-clamp-2 text-sm leading-snug font-semibold break-words">
                    {topic.title}
                  </h3>
                  <p className="text-faint mt-0.5 flex items-center gap-1 text-xs break-words">
                    <SkillIcon skill={topic.skill} size={12} />
                    {t(`skill.${topic.skill}`)}
                  </p>
                </div>
                <Badge
                  variant={statusBadge[topic.status].variant}
                  className={cn(
                    badgeShape,
                    "font-medium",
                    statusBadge[topic.status].className,
                  )}
                >
                  {t(`topicStatus.${topic.status}`)}
                </Badge>
                {open ? (
                  <IconChevronUp size={13} className="text-faint shrink-0" />
                ) : (
                  <IconChevronRight size={13} className="text-faint shrink-0" />
                )}
                {topic.status !== "completed" && topic.exercises.total > 0 && (
                  <span
                    role="progressbar"
                    aria-label={t("topicExercisesDone", topic.exercises)}
                    aria-valuemin={0}
                    aria-valuemax={topic.exercises.total}
                    aria-valuenow={topic.exercises.done}
                    className="bg-border absolute inset-x-0 bottom-0 h-0.5"
                  >
                    <span
                      className="bg-success block h-full transition-[width] duration-300 motion-reduce:transition-none"
                      style={{
                        width: `${(topic.exercises.done / topic.exercises.total) * 100}%`,
                      }}
                    />
                  </span>
                )}
              </button>
              {open && <TopicRunner topicId={topic.id} />}
            </div>
            </div>
          )
        })}
      </section>
    </div>
  )
}
