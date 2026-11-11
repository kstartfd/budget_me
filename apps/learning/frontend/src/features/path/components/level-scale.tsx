"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { cn } from "@/lib/utils"

const LEVELS = ["A1", "A2", "B1", "B2", "C1", "C2"] as const

/** How long the "A2 passed" pill and the "new stage" caption stay. */
const toastMs = 1800
/** How long the finished rung stays enlarged. */
const popMs = 450
/** How long the freshly filled tick bounces. */
const tickMs = 450
/** The ticks fill and fade in a wave, this many ms apart. */
const waveMs = 30
/** The last stretch is where tiredness turns into clicking at random, so the
 * line changes tone once this few questions are left. */
const almostLeft = 2

type Step = "done" | "cur" | "fin" | "up"
type Mode = "norm" | "push" | "done"

type LevelScaleProps =
  | { mode: "placement"; level: string; answered: number; total: number }
  | { mode: "final"; level: string; answered: number; total: number }
  | { mode: "determined"; level: string }

/** The colours of the achievement line: green, and a warm amber for the last
 * stretch. Never red: a mistake is not penalised here. */
const tones = {
  norm: {
    text: "text-success",
    fill: "bg-success border-success",
    border: "border-border",
    chip: "bg-success/15 text-success",
  },
  push: {
    text: "text-[#b45309] dark:text-[#fbbf24]",
    fill: "bg-[#b45309] border-[#b45309] dark:bg-[#fbbf24] dark:border-[#fbbf24]",
    border: "border-[#b45309] dark:border-[#fbbf24]",
    chip: "bg-[#fef3c7] text-[#b45309] dark:bg-[#3a2a0a] dark:text-[#fbbf24]",
  },
  done: {
    text: "text-success",
    fill: "bg-success border-success",
    border: "border-success",
    chip: "bg-success/15 text-success",
  },
} as const

/**
 * LevelScale is the ladder of a level test: the rungs A1..C2 with what is
 * passed, where the learner is now and what is still ahead, and under it the
 * achievement line of the level in hand: one tick per question, how many are
 * left and a nudge that warms up as the end of the level comes close. Nobody
 * knows in advance where the examiner will stop, so it never promises a fixed
 * end: the whole ladder is shown and the caption says the test goes on while
 * it goes well. State is carried by shape and glyph as well as colour.
 */
export function LevelScale(props: LevelScaleProps) {
  const t = useTranslations("path.test")
  const index = Math.max(LEVELS.indexOf(props.level as (typeof LEVELS)[number]), 0)
  const level = LEVELS[index]

  const [pop, setPop] = useState(-1)
  const [passed, setPassed] = useState<string | null>(null)
  const [fresh, setFresh] = useState(-1)
  const previousLevel = useRef(level)
  const previousAnswered = useRef(0)

  const placement = props.mode === "placement"
  const answered = props.mode === "determined" ? 0 : props.answered
  const total = props.mode === "determined" ? 0 : props.total

  // Climbing a rung: the finished one swells for a moment, a pill says it is
  // passed, the line resets as a new stage, and a phone gives a light tap.
  useEffect(() => {
    const before = previousLevel.current
    previousLevel.current = level

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

  // Each answer fills the next tick, and that tick bounces once.
  useEffect(() => {
    const before = previousAnswered.current
    previousAnswered.current = answered

    if (!placement || answered <= before || answered >= total) return

    setFresh(answered - 1)
    const stop = setTimeout(() => setFresh(-1), tickMs)

    return () => clearTimeout(stop)
  }, [answered, total, placement])

  if (props.mode === "final") {
    const status = t("scaleFinalStatus", { level, n: answered, total })

    return (
      <div className="border-border bg-background sticky top-2 z-10 space-y-3 rounded-2xl border p-3 lg:p-5">
        <div role="img" aria-label={status} className="flex gap-1">
          {Array.from({ length: total }, (_, i) => (
            <span
              key={i}
              className={cn(
                "box-border h-3 flex-1 rounded-md border-[1.5px] transition-colors duration-[400ms] lg:h-3.5 motion-reduce:transition-none",
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

  const stepOf = (i: number): Step =>
    i < index ? "done" : i === index ? (placement ? "cur" : "fin") : "up"

  const left = Math.max(total - answered, 0)
  const mode: Mode =
    !placement || total === 0
      ? "norm"
      : left === 0
        ? "done"
        : left <= almostLeft && answered > 0
          ? "push"
          : "norm"
  const tone = tones[mode]
  const accented = placement && mode !== "norm"
  const wave = mode === "done" || pop >= 0

  const status = placement ? t("scaleStatus", { level, n: answered, total }) : ""
  const announce = passed
    ? t("srPassed", { passed, level, total })
    : placement
      ? status
      : t("srDetermined", { level })

  return (
    <div
      className={cn(
        "bg-background sticky top-2 z-10 space-y-3 rounded-2xl border p-3 transition-colors duration-[400ms] lg:p-5 motion-reduce:transition-none",
        accented ? tone.border : "border-border",
      )}
    >
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

      {placement && total > 0 && (
        <>
          <div aria-hidden="true" className="mt-0.5 flex items-center gap-2">
            <span
              className={cn(
                "rounded-lg px-2 py-0.5 text-xs font-semibold whitespace-nowrap transition-colors duration-[400ms] motion-reduce:transition-none",
                tone.chip,
              )}
            >
              {passed ? `${passed} → ${level}` : level}
            </span>
            <div className="flex flex-1 items-center gap-1">
              {Array.from({ length: total }, (_, i) => {
                const on = i < answered
                const delay = wave ? (mode === "done" ? i : total - 1 - i) * waveMs : 0

                return (
                  <span
                    key={i}
                    style={{ transitionDelay: `${delay}ms` }}
                    className={cn(
                      "box-border h-2 flex-1 rounded-[5px] border-[1.5px] transition-all duration-[350ms] lg:h-2.5 motion-reduce:transition-none",
                      on ? `${tone.fill} border-solid` : "border-border border-dashed bg-transparent",
                      fresh === i && "motion-safe:animate-[tick-pop_450ms_ease]",
                      mode === "done" && "shadow-[0_0_0_3px_var(--success-wash)]",
                    )}
                  />
                )
              })}
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-x-2.5 gap-y-1.5">
            <span className="text-muted-foreground text-[13px]">
              {mode === "done"
                ? t("scaleReady")
                : t.rich("scaleLeft", {
                    n: left,
                    num: (chunks) => (
                      <span
                        className={cn(
                          "mx-px text-lg font-semibold tabular-nums transition-colors duration-[400ms] lg:text-xl motion-reduce:transition-none",
                          tone.text,
                        )}
                      >
                        {chunks}
                      </span>
                    ),
                  })}
            </span>
            {mode !== "norm" && (
              <span
                className={cn(
                  "relative rounded-full px-2.5 py-[3px] text-xs font-semibold transition-colors duration-[400ms] motion-reduce:transition-none",
                  tone.chip,
                )}
              >
                {mode === "push" ? t("scaleAlmost") : t("scaleAll")}
                {mode === "done" && (
                  <>
                    <Spark className="-top-1 left-1.5" delay={0} />
                    <Spark className="-top-1 right-2" delay={200} />
                  </>
                )}
              </span>
            )}
          </div>
        </>
      )}

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
        {passed ? t("scaleNewStage") : t(placement ? "scaleCaption" : "scaleDoneCaption")}
      </p>

      <p className="sr-only" aria-live="polite">
        {announce}
      </p>
    </div>
  )
}

/** Two small sparks float up from the "all answered" badge, once. */
function Spark({ className, delay }: { className: string; delay: number }) {
  return (
    <span
      aria-hidden="true"
      style={{ animationDelay: `${delay}ms` }}
      className={cn(
        "pointer-events-none absolute text-[10px] opacity-0 motion-safe:animate-[spark_1s_ease_1_forwards]",
        className,
      )}
    >
      ✦
    </span>
  )
}
