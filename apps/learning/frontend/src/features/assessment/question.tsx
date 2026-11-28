"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { card } from "@/features/home/components/parts"
import type { PathQuestion } from "@/features/path/schemas"
import { cn } from "@/lib/utils"

export function AssessmentQuestion({
  question,
  language,
  locale,
  answered,
  resumedFrom,
  busy,
  error,
  onAnswer,
  onPause,
}: {
  question: PathQuestion
  language: string
  locale: string
  answered: number
  resumedFrom: number | null
  busy: boolean
  error?: string
  onAnswer: (answer: string) => void
  onPause: () => void
}) {
  const t = useTranslations("assessment")
  const [answer, setAnswer] = useState("")
  const title = useRef<HTMLHeadingElement>(null)
  const type =
    question.type ??
    (question.options.length ? "multiple_choice" : "open_gap_fill")
  const correction = type === "error_correction"

  useEffect(() => {
    window.scrollTo(0, 0)
    title.current?.focus({ preventScroll: true })
  }, [])

  return (
    <div className="mx-auto w-full max-w-[720px] space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-muted-foreground text-xs font-medium">
            {t("activeTitle", { language })}
          </h2>
          <p className="text-muted-foreground text-xs" aria-live="polite">
            {t("questionProgress", { question: answered + 1, answered })}
          </p>
        </div>
        <Button
          variant="outline"
          className="h-11 rounded-lg px-4 text-sm"
          disabled={busy}
          onClick={onPause}
        >
          {t("saveExit")}
        </Button>
      </header>
      {resumedFrom === answered && answered > 0 && (
        <p
          role="status"
          className="bg-info/10 text-info rounded-lg px-4 py-3 text-sm"
        >
          {t("resumed", { count: answered })}
        </p>
      )}
      <form
        className={cn(card, "space-y-5 p-4 sm:p-6")}
        aria-labelledby="assessment-question"
        aria-busy={busy}
        onSubmit={(event) => {
          event.preventDefault()
          if (!busy && answer.trim()) onAnswer(answer.trim())
        }}
      >
        <div className="space-y-2">
          <p className="text-muted-foreground text-xs font-medium">
            {t("skill.grammar")} · {t(`format.${type}`)}
          </p>
          {!correction && (
            <p className="text-muted-foreground text-sm">{t(`task.${type}`)}</p>
          )}
          {question.text && (
            <p
              lang={locale}
              className="bg-muted rounded-lg px-3.5 py-3 text-base"
            >
              {question.text}
            </p>
          )}
          <h2
            id="assessment-question"
            ref={title}
            tabIndex={-1}
            lang={correction ? undefined : locale}
            className="text-xl font-semibold leading-snug wrap-break-word focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring"
          >
            {correction
              ? t("correctWord", { word: question.prompt })
              : question.prompt}
          </h2>
        </div>
        {question.options.length > 0 ? (
          <fieldset
            className="grid gap-2"
            disabled={busy}
            aria-labelledby="assessment-question"
          >
            {question.options.map((option) => (
              <label
                key={option}
                className={cn(
                  "border-border flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border px-3.5 py-2.5 text-sm has-focus-visible:ring-2 has-focus-visible:ring-ring",
                  answer === option &&
                    "border-foreground bg-muted ring-1 ring-foreground",
                )}
              >
                <input
                  type="radio"
                  name="assessment-answer"
                  className="accent-foreground size-4 shrink-0"
                  checked={answer === option}
                  onChange={() => setAnswer(option)}
                />
                <span lang={locale} className="min-w-0 wrap-break-word">
                  {option}
                </span>
              </label>
            ))}
          </fieldset>
        ) : (
          <label className="block space-y-1.5 text-sm">
            <span>{t("yourAnswer")}</span>
            <Input
              lang={locale}
              value={answer}
              onChange={(event) => setAnswer(event.target.value)}
              className="h-11 text-base"
              maxLength={200}
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              disabled={busy}
            />
          </label>
        )}
        {error && (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        )}
        <div className="border-border flex flex-wrap items-center justify-between gap-2 border-t pt-4">
          <Button
            type="button"
            variant="outline"
            className="h-11 rounded-lg px-4 text-sm"
            disabled={busy}
            onClick={() => onAnswer("")}
          >
            {t("dontKnow")}
          </Button>
          <Button
            type="submit"
            className="h-11 rounded-lg px-4 text-sm"
            disabled={busy || !answer.trim()}
          >
            {t(busy ? "saving" : "continue")}
          </Button>
        </div>
      </form>
      <p className="text-muted-foreground text-xs">{t("answerNote")}</p>
    </div>
  )
}
