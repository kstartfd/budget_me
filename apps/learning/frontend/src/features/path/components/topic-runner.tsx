"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconCheck, IconMessageCircle, IconX } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { useLearning } from "@/features/learning/hooks"
import type { Topic } from "@/features/learning/schemas"
import { useTopicDetail, useTopicExerciseAttempt } from "../hooks"
import type { PathAttempt, PathExercise } from "../schemas"

const baseLanguage = (code: string) => code.split("-")[0].toLowerCase()

/** The notebook whose language pair is the program's: the chat with the
 * teacher is kept per notebook, so this is the one to open it on. */
function chatTopicFor(
  topics: Topic[] | undefined,
  language: string,
  translationLanguage: string,
): Topic | undefined {
  const sameLanguage = topics?.filter(
    (item) => baseLanguage(item.language) === baseLanguage(language),
  )

  return (
    sameLanguage?.find(
      (item) =>
        baseLanguage(item.translationLanguage) ===
        baseLanguage(translationLanguage),
    ) ?? sameLanguage?.[0]
  )
}

/** TopicRunner is the study screen of one topic: its exercises, checked one
 * answer at a time, with the agent's explanation when an answer is wrong. */
export function TopicRunner({ topicId }: { topicId: string }) {
  const t = useTranslations("path.topic")
  const { detail, generate, complete } = useTopicDetail(topicId)
  const { topics } = useLearning()

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

  const { path, topic, exercises, attempts } = detail.data
  const chatTopic = chatTopicFor(
    topics.data,
    path.language,
    path.translationLanguage,
  )
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
      <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3">
        <p className="text-muted-foreground text-[10px] font-semibold tracking-wider uppercase">
          {t("rule")}
        </p>
        <p className="text-[13px] leading-relaxed">{topic.goal}</p>
      </div>
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
          {exercises.map((exercise, index) => (
            <ExerciseCard
              key={exercise.id}
              topicId={topicId}
              exercise={exercise}
              index={index + 1}
              total={exercises.length}
              topicTitle={topic.title}
              chatTopicId={chatTopic?.id}
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
  index,
  total,
  topicTitle,
  chatTopicId,
  solved,
  topicId,
  latestAttempt,
}: {
  topicId: string
  exercise: PathExercise
  index: number
  total: number
  topicTitle: string
  chatTopicId?: string
  solved: boolean
  latestAttempt?: PathAttempt
}) {
  const t = useTranslations("path.topic")
  const attempt = useTopicExerciseAttempt(topicId)
  const [value, setValue] = useState("")
  const [last, setLast] = useState<PathAttempt | null>(null)
  const result = last ?? latestAttempt
  const chat = new URLSearchParams({
    question: t("chatQuestion", { topic: topicTitle, prompt: exercise.prompt }),
  })
  if (chatTopicId) chat.set("topic", chatTopicId)

  const submit = (answer: string) => {
    if (!answer.trim()) return
    attempt.mutate(
      { exerciseId: exercise.id, answer },
      { onSuccess: (result) => setLast(result) },
    )
  }

  const done = result ? result.correct : solved
  const failed = !!result && !result.correct

  return (
    <section className="border-border space-y-3 border-t pt-4 first-of-type:border-t-0 first-of-type:pt-0">
      {exercise.text && (
        <p className="text-muted-foreground text-sm leading-relaxed italic">
          {exercise.text}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="text-muted-foreground text-xs tabular-nums">
          {index} / {total}
        </span>
        <p className="text-sm leading-snug font-semibold">{exercise.prompt}</p>
        {(done || failed) && (
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-[11px] font-medium",
              done
                ? "border-success/30 bg-success/10 text-success"
                : "border-destructive/30 bg-destructive/10 text-destructive",
            )}
          >
            {t(done ? "solved" : "failed")}
          </span>
        )}
      </div>

      {exercise.kind === "choice" ? (
        <div className="grid gap-2">
          {exercise.options.map((option) => {
            const picked = result?.answer === option
            const right = picked && result.correct
            const wrong = picked && !result.correct
            const reveal =
              failed && !!exercise.answer && option === exercise.answer

            return (
              <button
                key={option}
                type="button"
                onClick={() => submit(option)}
                disabled={attempt.isPending}
                className={cn(
                  "border-input bg-background hover:bg-muted/50 flex min-h-10 w-full cursor-pointer items-center gap-3 rounded-md border px-3.5 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed",
                  right && "border-success bg-success/10 text-success font-semibold",
                  wrong &&
                    "border-destructive bg-destructive/10 text-destructive font-semibold",
                  reveal && "border-success text-success border-dashed font-semibold",
                )}
              >
                <span
                  className={cn(
                    "border-input size-4 shrink-0 rounded-full border",
                    right && "border-success bg-success",
                    wrong && "border-destructive bg-destructive",
                    reveal && "border-success",
                  )}
                />
                <span className="min-w-0 flex-1">{option}</span>
                {(right || reveal) && (
                  <IconCheck size={14} className="shrink-0" />
                )}
                {wrong && <IconX size={14} className="shrink-0" />}
              </button>
            )
          })}
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
          className={cn(
            "space-y-1 rounded-lg border px-3.5 py-3 text-sm",
            result.correct
              ? "border-success/30 bg-success/10"
              : "border-destructive/30 bg-destructive/10",
          )}
        >
          <p
            className={cn(
              "font-semibold",
              result.correct ? "text-success" : "text-destructive",
            )}
          >
            {result.correct ? t("correct") : t("wrong")}
          </p>
          <p className="text-foreground text-[13px] leading-relaxed">
            {t("myAnswer")}: {result.answer}.
            {!result.correct && result.feedback && (
              <>
                {" "}
                <QuotedEmphasis text={result.feedback} />
              </>
            )}
          </p>
          {!result.correct && (
            <Button
              asChild
              variant="outline"
              size="lg"
              className="bg-background mt-2 text-[13px]"
            >
              <Link href={`/learning/chat?${chat}`}>
                <IconMessageCircle size={14} />
                {t("askTeacher")}
              </Link>
            </Button>
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
