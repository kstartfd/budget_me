"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { cn } from "@/lib/utils"

const LEVELS = ["A1", "A2", "B1", "B2", "C1", "C2"] as const

/** How long the "A2 passed" pill stays after the learner climbs a rung. */
const toastMs = 1800
/** How long the finished rung stays enlarged. */
const popMs = 450

type Step = "done" | "cur" | "fin" | "up"

type LevelScaleProps =
  | { mode: "placement"; level: string; answered: number; total: number }
  | { mode: "final"; level: string; answered: number; total: number }
  | { mode: "determined"; level: string }

/**
 * LevelScale is the ladder of a level test: the rungs A1..C2 with what is
 * passed, where the learner is now and what is still ahead. Nobody knows in
 * advance where the examiner will stop, so it never promises a fixed end: the
 * whole ladder is shown and the caption says the test goes on while it goes
 * well. State is carried by shape and glyph as well as colour: a check for a
 * passed rung, "n/8" for the current one, a dashed outline for what is ahead
 * and a star for the determined level.
 */
export function LevelScale(props: LevelScaleProps) {
  const t = useTranslations("path.test")
  const index = Math.max(LEVELS.indexOf(props.level as (typeof LEVELS)[number]), 0)
  const level = LEVELS[index]

  const [pop, setPop] = useState(-1)
  const [passed, setPassed] = useState<string | null>(null)
  const previous = useRef(level)

  const placement = props.mode === "placement"

  // Climbing a rung: the finished one swells for a moment, a pill says it is
  // passed, and a phone gives a light tap.
  useEffect(() => {
    const before = previous.current
    previous.current = level

    if (!placement || before === level || LEVELS.indexOf(before) > index) return

    setPop(LEVELS.indexOf(before))
    setPassed(before)
    if (typeof navigator !== "undefined") navigator.vibrate?.(10)

    const stopPop = setTimeout(() => setPop(-1), popMs)
    const stopToast = setTimeout(() => setPassed(null), toastMs)

    return () => {
      clearTimeout(stopPop)
      clearTimeout(stopToast)
    }
  }, [level, index, placement])

  if (props.mode === "final") {
    const { answered, total } = props
    const status = t("scaleFinalStatus", { level, n: answered, total })

    return (
      <div className="border-border bg-background sticky top-2 z-10 space-y-3 rounded-2xl border p-3 lg:p-5">
        <div role="img" aria-label={status} className="flex gap-1">
          {Array.from({ length: total }, (_, i) => (
            <span
              key={i}
              className={cn(
                "h-3 flex-1 rounded-md border-[1.5px] transition-colors duration-[400ms] motion-reduce:transition-none lg:h-3.5",
                i < answered
                  ? "border-success bg-success border-solid"
                  : "border-border border-dashed bg-transparent",
              )}
            />
          ))}
        </div>
        <p className="text-sm font-semibold lg:text-[15px]">{status}</p>
        <p className="text-muted-foreground text-[13px] leading-[18px] text-pretty">
          {t("scaleFinalCaption")}
        </p>
      </div>
    )
  }

  const total = props.mode === "placement" ? props.total : 0
  const answered = props.mode === "placement" ? props.answered : 0

  const stepOf = (i: number): Step =>
    i < index ? "done" : i === index ? (placement ? "cur" : "fin") : "up"

  const status = placement ? t("scaleStatus", { level, n: answered, total }) : ""
  const announce = passed
    ? t("srPassed", { passed, level, total })
    : placement
      ? status
      : t("srDetermined", { level })

  return (
    <div className="border-border bg-background sticky top-2 z-10 space-y-3 rounded-2xl border p-3 lg:p-5">
      <ol aria-label={t("scaleLabel")} className="m-0 flex list-none gap-1.5 p-0">
        {LEVELS.map((name, i) => {
          const step = stepOf(i)
          const label = {
            done: t("stepDone"),
            cur: t("stepCurrent"),
            fin: t("stepDetermined"),
            up: t("stepNext"),
          }[step]

          return (
            <li
              key={name}
              aria-label={`${name}, ${label}`}
              aria-current={step === "cur" || step === "fin" ? "step" : undefined}
              className={cn(
                "relative box-border flex h-14 min-w-0 flex-1 flex-col items-center justify-center gap-0.5 overflow-hidden rounded-xl transition-all duration-[400ms] lg:h-16 motion-reduce:transition-none",
                pop === i && "motion-safe:scale-[1.08]",
                step === "done" && "border-success bg-success border-[1.5px]",
                step === "cur" && "border-success bg-success/15 border-2",
                step === "fin" &&
                  "border-success bg-success ring-success/20 border-[1.5px] ring-[3px]",
                step === "up" && "border-border border-[1.5px] border-dashed bg-transparent",
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "text-[11px] leading-[14px] font-semibold tabular-nums",
                  step === "cur" && "text-success",
                  step === "up" && "text-muted-foreground",
                  (step === "done" || step === "fin") && "text-white dark:text-zinc-950",
                )}
              >
                {step === "done" ? "✓" : step === "fin" ? "★" : step === "cur" ? `${answered}/${total}` : "–"}
              </span>
              <span
                aria-hidden="true"
                className={cn(
                  "text-[15px] leading-[18px] font-semibold lg:text-base",
                  step === "cur" && "text-success",
                  step === "up" && "text-muted-foreground",
                  (step === "done" || step === "fin") && "text-white dark:text-zinc-950",
                )}
              >
                {name}
              </span>
              {step === "cur" && (
                <span
                  aria-hidden="true"
                  className="bg-border absolute right-[7px] bottom-1.5 left-[7px] block h-[3px] overflow-hidden rounded-sm"
                >
                  <span
                    className="bg-success block h-full rounded-sm transition-[width] duration-[400ms] motion-reduce:transition-none"
                    style={{ width: `${total > 0 ? (answered / total) * 100 : 0}%` }}
                  />
                </span>
              )}
            </li>
          )
        })}
      </ol>

      {placement && (
        <div className="flex min-h-[22px] items-center justify-between gap-2">
          <span className="text-sm font-semibold lg:text-[15px]">{status}</span>
          <span
            aria-hidden="true"
            className={cn(
              "bg-success/15 text-success rounded-full px-2.5 py-0.5 text-xs font-semibold transition-opacity duration-500 motion-reduce:transition-none",
              passed ? "opacity-100" : "opacity-0",
            )}
          >
            {t("levelPassed", { level: passed ?? level })}
          </span>
        </div>
      )}

      <p className="text-muted-foreground text-[13px] leading-[18px] text-pretty">
        {t(placement ? "scaleCaption" : "scaleDoneCaption")}
      </p>

      <p className="sr-only" aria-live="polite">
        {announce}
      </p>
    </div>
  )
}
