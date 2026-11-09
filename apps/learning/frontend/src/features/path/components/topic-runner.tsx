"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useTopicDetail, useTopicExerciseAttempt } from "../hooks"
import type { PathAttempt, PathExercise } from "../schemas"

/** TopicRunner is the study screen of one topic: its exercises, checked one
 * answer at a time, with the agent's explanation when an answer is wrong. */
export function TopicRunner({ topicId }: { topicId: string }) {
  const t = useTranslations("path.topic")
  const { detail, generate, complete } = useTopicDetail(topicId)

  if (detail.isPending)
    return (
      <p className="text-muted-foreground py-6 text-center text-sm">
        {t("loading")}
      </p>
    )
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
  const currentExerciseIds = new Set(exercises.map((exercise) => exercise.id))
  const pastAttempts = attempts.filter(
    (attempt) => !currentExerciseIds.has(attempt.exerciseId),
  )

  return (
    <div className="bg-secondary/40 space-y-3 rounded-[22px] border border-transparent p-4">
      <p className="text-muted-foreground text-sm leading-relaxed">
        {topic.goal}
      </p>
      {pastAttempts.length > 0 && (
        <p className="text-muted-foreground text-xs">
          {t("pastAttempts", {
            count: pastAttempts.length,
            correct: pastAttempts.filter((attempt) => attempt.correct).length,
          })}
        </p>
      )}

      {generate.isError && (
        <p role="alert" className="text-destructive text-sm">
          {generate.error.message}
        </p>
      )}

      {exercises.length === 0 ? null : (
        <>
          {exercises.map((exercise) => (
            <ExerciseCard
              key={exercise.id}
              topicId={topicId}
              exercise={exercise}
              solved={solved.has(exercise.id)}
            />
          ))}

          <div className="space-y-2 pt-1">
            <TopicActions
              hasExercises
              generating={generate.isPending}
              onGenerate={() => generate.mutate(false)}
              onReset={() => generate.mutate(true)}
            />
            {topic.status !== "completed" && (
              <>
                {complete.isError && (
                  <p role="alert" className="text-destructive text-sm">
                    {complete.error.message}
                  </p>
                )}
                <Button
                  variant="outline"
                  onClick={() => complete.mutate()}
                  disabled={complete.isPending || generate.isPending}
                  className="border-primary/30 text-primary h-auto w-full rounded-[18px] py-3 text-sm font-semibold"
                >
                  {t(complete.isPending ? "completing" : "complete")}
                </Button>
              </>
            )}
          </div>
        </>
      )}
      {exercises.length === 0 && (
        <TopicActions
          generating={generate.isPending}
          onGenerate={() => generate.mutate(false)}
          onReset={() => generate.mutate(true)}
        />
      )}
    </div>
  )
}

function TopicActions({
  hasExercises = false,
  generating,
  onGenerate,
  onReset,
}: {
  hasExercises?: boolean
  generating: boolean
  onGenerate: () => void
  onReset: () => void
}) {
  const t = useTranslations("path.topic")
  return (
    <Button
      variant={hasExercises ? "outline" : "default"}
      onClick={hasExercises ? onReset : onGenerate}
      disabled={generating}
      className={`${hasExercises ? "text-primary" : "text-primary-foreground"} h-auto w-full rounded-[18px] py-3 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]`}
    >
      {t(
        generating
          ? "generating"
          : hasExercises
            ? "resetExercises"
            : "generate",
      )}
    </Button>
  )
}

function ExerciseCard({
  exercise,
  solved,
  topicId,
}: {
  topicId: string
  exercise: PathExercise
  solved: boolean
}) {
  const t = useTranslations("path.topic")
  const attempt = useTopicExerciseAttempt(topicId)
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
            placeholder={t(
              exercise.kind === "fill"
                ? "fillPlaceholder"
                : "translatePlaceholder",
            )}
            aria-label={exercise.prompt}
            className="bg-secondary border-border focus-visible:border-primary h-auto rounded-2xl px-4 py-3 text-[15px] focus-visible:ring-0"
          />
          <Button
            type="submit"
            disabled={attempt.isPending || !value.trim()}
            className="text-primary-foreground h-auto rounded-2xl px-5 text-sm font-bold"
          >
            {t(attempt.isPending ? "checking" : "check")}
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
            <p className="text-foreground leading-relaxed">
              <QuotedEmphasis text={last.feedback} />
            </p>
          )}
        </div>
      )}
    </section>
  )
}

function QuotedEmphasis({ text }: { text: string }) {
  const parts = text.split(/(["“«])([^"”»\n]+)(["”»])/g)

  return (
    <>
      {parts.map((part, index) => {
        if (index % 4 === 2) return <strong key={index}>{part}</strong>
        return part
      })}
    </>
  )
}
