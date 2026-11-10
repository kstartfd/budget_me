"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconCheck, IconMessageCircle } from "@tabler/icons-react"
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
  const latestAttempts = new Map<string, PathAttempt>()
  for (const attempt of attempts) {
    latestAttempts.set(attempt.exerciseId, attempt)
  }

  return (
    <div className="space-y-3 p-4 pt-4">
      <p className="text-muted-foreground text-[13px] leading-relaxed">
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
              topicGoal={topic.goal}
              solved={solved.has(exercise.id)}
              latestAttempt={latestAttempts.get(exercise.id)}
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
                  size="lg"
                  onClick={() => complete.mutate()}
                  disabled={complete.isPending || generate.isPending}
                  className="w-full text-[13px]"
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
      size="lg"
      onClick={hasExercises ? onReset : onGenerate}
      disabled={generating}
      className="w-full text-[13px]"
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
  topicGoal,
  solved,
  topicId,
  latestAttempt,
}: {
  topicId: string
  exercise: PathExercise
  topicGoal: string
  solved: boolean
  latestAttempt?: PathAttempt
}) {
  const t = useTranslations("path.topic")
  const attempt = useTopicExerciseAttempt(topicId)
  const [value, setValue] = useState("")
  const [last, setLast] = useState<PathAttempt | null>(null)
  const result = last ?? latestAttempt
  const rule = exercise.rule || topicGoal
  const chatQuestion = t("chatQuestion", {
    rule,
    prompt: exercise.prompt,
  })

  const submit = (answer: string) => {
    if (!answer.trim()) return
    attempt.mutate(
      { exerciseId: exercise.id, answer },
      { onSuccess: (result) => setLast(result) },
    )
  }

  return (
    <section className="border-border space-y-3 border-t pt-4 first-of-type:border-t-0 first-of-type:pt-0">
      <aside className="bg-muted space-y-2.5 rounded-lg p-4">
        <div className="space-y-1">
          <p className="text-muted-foreground text-xs font-semibold">
            {t("rule")}
          </p>
          <p className="text-[13px] leading-relaxed">{rule}</p>
        </div>
        {result?.correct && (
          <div className="border-border space-y-1 border-t pt-2.5">
            <p className="text-muted-foreground text-xs font-semibold">
              {t("correctAnswer")}
            </p>
            <p className="text-success text-sm font-semibold">
              {result.answer}
            </p>
          </div>
        )}
        <Button
          asChild
          variant="outline"
          size="lg"
          className="bg-background h-8 px-3 text-xs"
        >
          <Link href={`/learning/chat?question=${encodeURIComponent(chatQuestion)}`}>
            <IconMessageCircle size={13} />
            {t("askTeacher")}
          </Link>
        </Button>
      </aside>
      {exercise.text && (
        <p className="text-muted-foreground text-sm leading-relaxed italic">
          {exercise.text}
        </p>
      )}
      <p className="text-sm leading-snug font-semibold">
        {exercise.prompt}
        {solved && (
          <span className="bg-success/10 text-success ml-2 rounded-md px-2 py-0.5 align-middle text-[11px] font-semibold">
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
              className={`border-border hover:bg-muted/50 flex min-h-10 cursor-pointer items-center justify-between gap-3 rounded-md border px-3.5 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed ${
                result?.answer === option && result.correct
                  ? "bg-muted font-semibold"
                  : "bg-background"
              }`}
            >
              {option}
              {result?.answer === option && result.correct && (
                <IconCheck size={13} className="text-success shrink-0" />
              )}
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
            className="bg-background h-10 text-sm md:text-sm"
          />
          <Button
            type="submit"
            disabled={attempt.isPending || !value.trim()}
            className="h-10 px-5 text-sm"
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
      {result && (
        <div
          className={`space-y-2 rounded-lg px-4 py-3 text-sm ${
            result.correct
              ? "bg-success/10 text-success"
              : "bg-destructive/10 text-destructive"
          }`}
        >
          <p className="font-semibold">
            {result.correct ? t("correct") : t("wrong")}
          </p>
          <div>
            <p className="text-xs font-semibold">{t("myAnswer")}</p>
            <p className="text-foreground mt-0.5 leading-relaxed">
              {result.answer}
            </p>
          </div>
          {!result.correct && result.feedback && (
            <p className="text-foreground leading-relaxed">
              <QuotedEmphasis text={result.feedback} />
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
