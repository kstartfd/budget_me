"use client"

import { useEffect, useRef, useState } from "react"
import { useFormatter, useTranslations } from "next-intl"
import { IconPlayerPause, IconPlayerPlay } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useLesson } from "../hooks"
import type { PathLesson } from "../schemas"

const clock = (seconds: number) => {
  const s = Math.max(0, Math.ceil(seconds))
  const mm = String(Math.floor(s / 60)).padStart(2, "0")
  const ss = String(s % 60).padStart(2, "0")
  return `${mm}:${ss}`
}

/** Seconds the learner was at the desk: the whole time, less the pauses. */
function studiedSeconds(lesson: PathLesson, now: number) {
  const end = lesson.finishedAt ? Date.parse(lesson.finishedAt) : now
  const paused =
    lesson.pausedSeconds +
    (lesson.pausedAt && !lesson.finishedAt
      ? Math.max(0, (now - Date.parse(lesson.pausedAt)) / 1000)
      : 0)
  return Math.max(0, (end - Date.parse(lesson.startedAt)) / 1000 - paused)
}

/** Counts down to a moment on the server's clock; the lesson never waits. */
function useRemaining(endsAt: string | undefined) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!endsAt) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [endsAt])
  return { now, left: endsAt ? (Date.parse(endsAt) - now) / 1000 : 0 }
}

/**
 * The 30-minute lesson of a topic: start it, watch the clock, step away (the
 * clock keeps going) and finish, with the result kept for later analysis.
 * `onPausedChange` lets the topic hide its exercises while the learner is away.
 */
export function LessonPanel({
  topicId,
  onPausedChange,
}: {
  topicId: string
  onPausedChange: (paused: boolean) => void
}) {
  const t = useTranslations("path.lesson")
  const format = useFormatter()
  const { lesson, history, start, change } = useLesson(topicId)
  const { now, left } = useRemaining(lesson?.endsAt)
  const [last, setLast] = useState<PathLesson | null>(null)
  const closing = useRef<string | null>(null)

  const paused = !!lesson?.pausedAt
  useEffect(() => {
    onPausedChange(paused)
  }, [paused, onPausedChange])

  // Time is up: close the lesson once; the server closes it at its end anyway.
  const id = lesson?.id
  const timeUp = !!lesson && left <= 0
  const { mutate: finish } = change
  useEffect(() => {
    if (!id || !timeUp || closing.current === id) return
    closing.current = id
    finish({ id, action: "finish" }, { onSuccess: setLast })
  }, [id, timeUp, finish])

  const finished = (history.data ?? []).filter((l) => l.status === "finished")
  const minutes = (seconds: number) => Math.max(1, Math.round(seconds / 60))

  if (!lesson) {
    return (
      <div className="space-y-3">
        {last && (
          <div className="border-success/30 bg-success/10 space-y-1 rounded-lg border px-3.5 py-3">
            <p className="text-success text-sm font-semibold">{t("finished")}</p>
            <p className="text-[13px]">
              {t("result", {
                minutes: minutes(studiedSeconds(last, now)),
                attempts: last.attempts,
                correct: last.correct,
              })}
            </p>
          </div>
        )}
        {start.isError && (
          <p role="alert" className="text-destructive text-sm">
            {start.error.message}
          </p>
        )}
        <Button
          size="lg"
          onClick={() => {
            setLast(null)
            start.mutate()
          }}
          disabled={start.isPending}
          className="w-full text-[13px]"
        >
          {t(start.isPending ? "starting" : "start")}
        </Button>
        {finished.length > 0 && (
          <details className="text-xs">
            <summary className="text-muted-foreground cursor-pointer">
              {t("history", { count: finished.length })}
            </summary>
            <ul className="mt-2 space-y-1">
              {finished.map((l) => (
                <li
                  key={l.id}
                  className="text-muted-foreground flex justify-between gap-3"
                >
                  <span>
                    {format.dateTime(new Date(l.startedAt), {
                      dateStyle: "medium",
                      timeStyle: "short",
                    })}
                  </span>
                  <span className="tabular-nums">
                    {t("historyRow", {
                      minutes: minutes(studiedSeconds(l, now)),
                      attempts: l.attempts,
                      correct: l.correct,
                    })}
                  </span>
                </li>
              ))}
            </ul>
          </details>
        )}
      </div>
    )
  }

  const total = Date.parse(lesson.endsAt) - Date.parse(lesson.startedAt)
  const elapsed = Math.min(1, Math.max(0, 1 - (left * 1000) / total))

  return (
    <div className="bg-muted space-y-3 rounded-lg px-3.5 py-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-muted-foreground text-[10px] font-semibold tracking-wider uppercase">
            {t(paused ? "paused" : "running")}
          </p>
          <p
            className="text-2xl font-semibold tabular-nums"
            role="timer"
            aria-label={t("left")}
          >
            {clock(left)}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="lg"
            disabled={change.isPending}
            onClick={() =>
              change.mutate({
                id: lesson.id,
                action: paused ? "resume" : "pause",
              })
            }
            className="bg-background gap-1.5 text-[13px]"
          >
            {paused ? (
              <IconPlayerPlay size={14} />
            ) : (
              <IconPlayerPause size={14} />
            )}
            {t(paused ? "resume" : "pause")}
          </Button>
          <Button
            variant="outline"
            size="lg"
            disabled={change.isPending}
            onClick={() =>
              change.mutate(
                { id: lesson.id, action: "finish" },
                { onSuccess: setLast },
              )
            }
            className="bg-background text-[13px]"
          >
            {t("finish")}
          </Button>
        </div>
      </div>
      <div className="bg-border h-1 overflow-hidden rounded-full">
        <div
          className="bg-foreground h-full transition-[width]"
          style={{ width: `${elapsed * 100}%` }}
        />
      </div>
      {paused && (
        <p className="text-muted-foreground text-xs">{t("pausedNote")}</p>
      )}
      {change.isError && (
        <p role="alert" className="text-destructive text-sm">
          {change.error.message}
        </p>
      )}
    </div>
  )
}
