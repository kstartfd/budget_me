"use client"

import { useState } from "react"
import type { UseMutationResult } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useTopicDetail } from "../hooks"
import type { PathAttempt, PathExercise } from "../schemas"

type AttemptMutation = UseMutationResult<
  PathAttempt,
  Error,
  { exerciseId: string; answer: string }
>

/** TopicRunner is the study screen of one topic: its exercises, checked one
 * answer at a time, with the agent's explanation when an answer is wrong. */
export function TopicRunner({ topicId }: { topicId: string }) {
  const t = useTranslations("path.topic")
  const { detail, generate, attempt, complete } = useTopicDetail(topicId)

  if (detail.isPending)
    return <p className="text-muted-foreground py-6 text-center text-sm">{t("loading")}</p>
  if (detail.isError)
    return (
      <p role="alert" className="text-destructive py-6 text-center text-sm">
        {detail.error.message}
      </p>
    )

  const { topic, exercises, attempts } = detail.data
  const solved = new Set(
    attempts.filter((a) => a.correct).map((a) => a.exerciseId),
  )

  return (
    <div className="bg-secondary/40 space-y-3 rounded-[22px] border border-transparent p-4">
      <p className="text-muted-foreground text-sm leading-relaxed">
        {topic.goal}
      </p>

      {exercises.length === 0 ? (
        <div className="space-y-2">
          {generate.isError && (
            <p role="alert" className="text-destructive text-sm">
              {generate.error.message}
            </p>
          )}
          <Button
            onClick={() => generate.mutate(false)}
            disabled={generate.isPending || topic.status === "completed"}
            className="text-primary-foreground h-auto w-full rounded-[18px] py-3 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
          >
            {t(generate.isPending ? "generating" : "generate")}
          </Button>
        </div>
      ) : (
        <>
          {exercises.map((exercise) => (
            <ExerciseCard
              key={exercise.id}
              exercise={exercise}
              solved={solved.has(exercise.id)}
              attempt={attempt}
            />
          ))}

          {topic.status !== "completed" && (
            <div className="space-y-2 pt-1">
              {complete.isError && (
                <p role="alert" className="text-destructive text-sm">
                  {complete.error.message}
                </p>
              )}
              <Button
                variant="outline"
                onClick={() => complete.mutate()}
                disabled={complete.isPending}
                className="border-primary/30 text-primary h-auto w-full rounded-[18px] py-3 text-sm font-semibold"
              >
                {t(complete.isPending ? "completing" : "complete")}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function ExerciseCard({
  exercise,
  solved,
  attempt,
}: {
  exercise: PathExercise
  solved: boolean
  attempt: AttemptMutation
}) {
  const t = useTranslations("path.topic")
  const [value, setValue] = useState("")
  const [last, setLast] = useState<PathAttempt | null>(null)

  const submit = (answer: string) => {
    if (!answer.trim()) return
    attempt.mutate(
      { exerciseId: exercise.id, answer },
      { onSuccess: (result) => setLast(result) },
    )
  }

  return (
    <section className="bg-card border-border space-y-3 rounded-[18px] border p-4">
      {exercise.text && (
        <p className="text-muted-foreground text-sm leading-relaxed italic">
          {exercise.text}
        </p>
      )}
      <p className="text-[15px] leading-snug font-semibold">
        {exercise.prompt}
        {solved && (
          <span className="text-primary ml-2 text-xs font-bold">
            {t("solved")}
          </span>
        )}
      </p>

      {exercise.kind === "choice" ? (
        <div className="grid gap-2">
          {exercise.options.map((option) => (
            <button
              key={option}
              type="button"
              onClick={() => submit(option)}
              disabled={attempt.isPending}
              className={`border-border hover:border-primary/40 rounded-2xl border px-4 py-3 text-left text-sm transition-colors ${
                last?.answer === option && last.correct
                  ? "border-primary bg-[color:var(--primary-wash)] text-primary font-semibold"
                  : "bg-secondary"
              }`}
            >
              {option}
            </button>
          ))}
        </div>
      ) : (
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            submit(value)
          }}
        >
          <Input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={t(exercise.kind === "fill" ? "fillPlaceholder" : "translatePlaceholder")}
            aria-label={exercise.prompt}
            className="bg-secondary border-border focus-visible:border-primary h-auto rounded-2xl px-4 py-3 text-[15px] focus-visible:ring-0"
          />
          <Button
            type="submit"
            disabled={attempt.isPending || !value.trim()}
            className="text-primary-foreground h-auto rounded-2xl px-5 text-sm font-bold"
          >
            {t("check")}
          </Button>
        </form>
      )}

      {attempt.isError && (
        <p role="alert" className="text-destructive text-sm">
          {attempt.error.message}
        </p>
      )}
      {last && (
        <div
          className={`space-y-1 rounded-2xl px-4 py-3 text-sm ${
            last.correct
              ? "bg-[color:var(--primary-wash)] text-primary"
              : "bg-destructive/10 text-destructive"
          }`}
        >
          <p className="font-semibold">
            {last.correct ? t("correct") : t("wrong")}
          </p>
          {!last.correct && last.feedback && (
            <p className="text-foreground leading-relaxed">{last.feedback}</p>
          )}
        </div>
      )}
    </section>
  )
}
