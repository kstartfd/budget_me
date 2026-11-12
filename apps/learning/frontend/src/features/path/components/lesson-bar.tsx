"use client"

import { useEffect, useRef, useState } from "react"
import { useFormatter, useTranslations } from "next-intl"
import { IconCoffee, IconPlayerPause, IconPlayerPlay } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useLesson } from "../hooks"
import type { PathLesson } from "../schemas"

/** Height of the running bar: the open topic's row sticks right under it. */
export const lessonBarHeight = 52

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

const minutes = (seconds: number) => Math.max(1, Math.round(seconds / 60))

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

/** A lesson that ran its full time was closed at its end, not by the learner. */
const ranOut = (lesson: PathLesson) =>
  !!lesson.finishedAt && Date.parse(lesson.finishedAt) >= Date.parse(lesson.endsAt)

/** How long after its end a lesson that ran out still gets its rest window. */
const restWindowMs = 15 * 60 * 1000
const seenKey = "learning:lesson-rest-seen"

/** The rest window of a lesson is shown once; the browser remembers which. */
function restSeen(id: string) {
  try {
    return localStorage.getItem(seenKey) === id
  } catch {
    return false
  }
}

function markRestSeen(id: string) {
  try {
    localStorage.setItem(seenKey, id)
  } catch {
    // a private window: the window may show again, which does no harm
  }
}

/**
 * The learner's 30-minute lesson, above the topics of the program: start it,
 * watch the clock while moving between topics (it keeps running at the top of
 * the screen), step away (the clock keeps going) and finish. When the time is
 * up a window says the lesson is over and it is time to rest.
 */
export function LessonBar() {
  const t = useTranslations("path.lesson")
  const format = useFormatter()
  const { lesson, history, start, change } = useLesson()
  const { now, left } = useRemaining(lesson?.endsAt)
  // How the last lesson the learner closed themselves went, shown until the next.
  const [last, setLast] = useState<PathLesson | null>(null)
  // The lesson whose time ran out: the rest window is about it.
  const [rest, setRest] = useState<PathLesson | null>(null)
  // The one whose window was closed here, in case the browser cannot remember it.
  const [dismissed, setDismissed] = useState<string | null>(null)
  const closing = useRef<string | null>(null)

  const paused = !!lesson?.pausedAt

  // Time is up: close the lesson once (the server closes it at its end anyway)
  // and tell the learner to rest.
  const id = lesson?.id
  const timeUp = !!lesson && left <= 0
  const { mutate: finish } = change
  useEffect(() => {
    if (!id || !timeUp || closing.current === id) return
    closing.current = id
    finish(
      { id, action: "finish" },
      {
        onSuccess: (closed) => {
          if (!restSeen(closed.id)) setRest(closed)
        },
      },
    )
  }, [id, timeUp, finish])

  // The app was closed when the time ran out: the first look at the lessons
  // afterwards still owes the learner that window, once.
  const asOf = history.dataUpdatedAt
  const newest = history.data?.[0]
  const owed =
    !lesson &&
    !!newest &&
    ranOut(newest) &&
    asOf - Date.parse(newest.finishedAt as string) < restWindowMs &&
    newest.id !== dismissed &&
    !restSeen(newest.id)
  const shown = rest ?? (owed ? newest : null) ?? null

  const closeRest = () => {
    if (shown) {
      markRestSeen(shown.id)
      setDismissed(shown.id)
    }
    setRest(null)
  }

  const finished = (history.data ?? []).filter((l) => l.status === "finished")

  const restWindow = (
    <Dialog open={shown !== null} onOpenChange={(open) => !open && closeRest()}>
      <DialogContent className="gap-4 rounded-lg p-6 text-center sm:max-w-sm">
        <DialogHeader className="items-center gap-3 text-center">
          <span className="bg-success/15 text-success flex size-12 items-center justify-center rounded-full">
            <IconCoffee size={24} aria-hidden="true" />
          </span>
          <DialogTitle className="text-lg font-semibold">{t("restTitle")}</DialogTitle>
          <DialogDescription className="text-[13px] leading-relaxed">
            {t("restBody")}
          </DialogDescription>
        </DialogHeader>
        {shown && (
          <p className="text-muted-foreground text-[13px]">
            {t("result", {
              minutes: minutes(studiedSeconds(shown, now)),
              attempts: shown.attempts,
              correct: shown.correct,
            })}
          </p>
        )}
        <Button size="lg" onClick={closeRest} className="text-[13px]">
          {t("restOk")}
        </Button>
      </DialogContent>
    </Dialog>
  )

  if (!lesson) {
    return (
      <section className="bg-background border-border space-y-3 rounded-lg border p-4">
        {restWindow}
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
        <div className="space-y-1">
          <h2 className="text-sm font-semibold">{t("idleTitle")}</h2>
          <p className="text-muted-foreground max-w-[65ch] text-[13px] leading-relaxed">
            {t("idleHint")}
          </p>
        </div>
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
          className="w-full text-[13px] sm:w-auto sm:px-5"
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
                <li key={l.id} className="text-muted-foreground flex justify-between gap-3">
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
      </section>
    )
  }

  const total = Date.parse(lesson.endsAt) - Date.parse(lesson.startedAt)
  const elapsed = Math.min(1, Math.max(0, 1 - (left * 1000) / total))

  return (
    <>
      {restWindow}
      {/* One short row that stays under the app bar while the topics scroll. */}
      <div
        style={{ height: lessonBarHeight }}
        className="bg-muted border-border sticky top-[61px] z-20 flex items-center justify-between gap-2 overflow-hidden rounded-lg border px-3.5 shadow-sm lg:top-0"
      >
        <div className="flex min-w-0 items-baseline gap-2.5">
          <p
            className="text-xl leading-none font-semibold tabular-nums"
            role="timer"
            aria-label={t("left")}
          >
            {clock(left)}
          </p>
          <p className="text-muted-foreground truncate text-xs font-semibold tracking-wider uppercase">
            {t(paused ? "paused" : "running")}
          </p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button
            variant="outline"
            disabled={change.isPending}
            onClick={() =>
              change.mutate({ id: lesson.id, action: paused ? "resume" : "pause" })
            }
            className="bg-background gap-1.5 px-3 text-[13px]"
          >
            {paused ? <IconPlayerPlay size={14} /> : <IconPlayerPause size={14} />}
            {t(paused ? "resume" : "pause")}
          </Button>
          <Button
            variant="outline"
            disabled={change.isPending}
            onClick={() =>
              change.mutate({ id: lesson.id, action: "finish" }, { onSuccess: setLast })
            }
            className="bg-background px-3 text-[13px]"
          >
            {t("finish")}
          </Button>
        </div>
        <div className="bg-border absolute inset-x-0 bottom-0 h-0.5" aria-hidden="true">
          <div className="bg-foreground h-full transition-[width]" style={{ width: `${elapsed * 100}%` }} />
        </div>
      </div>
      {change.isError && (
        <p role="alert" className="text-destructive text-sm">
          {change.error.message}
        </p>
      )}
    </>
  )
}

/**
 * The start of a lesson from inside a topic, so the learner need not scroll to
 * the top of the program for it. It shows only while no lesson runs.
 */
export function StartLesson() {
  const t = useTranslations("path.lesson")
  const { lesson, start } = useLesson()

  if (lesson) return null

  return (
    <div className="space-y-2">
      {start.isError && (
        <p role="alert" className="text-destructive text-sm">
          {start.error.message}
        </p>
      )}
      <Button
        size="lg"
        onClick={() => start.mutate()}
        disabled={start.isPending}
        className="w-full text-[13px]"
      >
        {t(start.isPending ? "starting" : "start")}
      </Button>
    </div>
  )
}
