"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronLeft } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useMutation } from "@tanstack/react-query"
import { analyzeTest, answerBatch } from "../api"
import { usePathTest } from "../hooks"
import type { PathTest } from "../schemas"

/**
 * Runs one level test batch by batch: the ladder climbs while the batches
 * pass, and a finished placement hands over to the analysis that builds the
 * program.
 */
export function TestRunner({ id, onClose }: { id: string; onClose: (analyzed: boolean) => void }) {
  const t = useTranslations("path.test")
  const query = usePathTest(id)
  const [picked, setPicked] = useState<Record<string, string>>({})
  const [test, setTest] = useState<PathTest | null>(null)
  const [analyzed, setAnalyzed] = useState(false)

  const submit = useMutation({
    mutationFn: () =>
      answerBatch(
        id,
        (test ?? query.data)!.questions.map((q) => ({
          key: q.key,
          answer: picked[q.key] ?? "",
        })),
      ),
    onSuccess: (next) => {
      setTest(next)
      setPicked({})
    },
  })

  if (query.isPending)
    return <p className="text-muted-foreground py-10 text-center text-sm">{t("loading")}</p>
  if (query.isError)
    return (
      <div className="space-y-3 py-10 text-center">
        <p role="alert" className="text-destructive text-sm">
          {query.error.message}
        </p>
        <Button variant="secondary" onClick={() => onClose(false)}>
          {t("exit")}
        </Button>
      </div>
    )

  const current = test ?? query.data
  if (!current) return null

  if (current.status === "done") {
    return (
      <Done
        test={current}
        onClose={onClose}
        onAnalyzed={() => setAnalyzed(true)}
      />
    )
  }

  const allAnswered = current.questions.every((q) => picked[q.key])

  return (
    <div className="anim-fade space-y-4 lg:max-w-[704px]">
      <header className="space-y-1">
        <h1 className="text-[22px] font-semibold tracking-[-0.01em] lg:text-2xl">
          {t(current.kind === "final" ? "finalTitle" : "placementTitle", {
            level: current.level,
          })}
        </h1>
        <p className="text-muted-foreground text-[13px] lg:text-sm">
          {t("progress", {
            correct: current.correct,
            asked: current.asked,
            needed: current.needed,
          })}
        </p>
      </header>

      <div className="space-y-3">
        {current.questions.map((q, index) => (
          <QuestionCard
            key={q.key}
            index={index}
            question={q}
            picked={picked[q.key]}
            onPick={(answer) => setPicked((prev) => ({ ...prev, [q.key]: answer }))}
          />
        ))}
      </div>

      {submit.isError && (
        <p role="alert" className="text-destructive text-sm">
          {submit.error.message}
        </p>
      )}

      <div className="flex gap-3">
        <Button variant="secondary" onClick={() => onClose(analyzed)}>
          <IconChevronLeft size={16} />
          {t("exit")}
        </Button>
        <Button
          onClick={() => submit.mutate()}
          size="lg"
          disabled={!allAnswered || submit.isPending}
          className="flex-1 text-[13px]"
        >
          {t(submit.isPending ? "sending" : "send")}
        </Button>
      </div>
    </div>
  )
}

function QuestionCard({
  index,
  question,
  picked,
  onPick,
}: {
  index: number
  question: PathTest["questions"][number]
  picked: string | undefined
  onPick: (answer: string) => void
}) {
  const t = useTranslations("path.test")
  return (
    <section className="bg-background border-border space-y-3 rounded-lg border p-5">
      {question.text && (
        <p className="text-muted-foreground text-sm leading-relaxed italic">
          {question.text}
        </p>
      )}
      <p className="text-sm leading-snug font-semibold">
        {index + 1}. {question.prompt}
      </p>
      <div className="grid gap-2">
        {question.options.map((option) => (
          <button
            key={option}
            type="button"
            onClick={() => onPick(option)}
            aria-pressed={picked === option}
            className={`border-border hover:bg-muted/50 min-h-10 cursor-pointer rounded-md border px-3.5 py-2 text-left text-sm transition-colors ${
              picked === option
                ? "bg-muted font-semibold"
                : "bg-background"
            }`}
          >
            {option}
          </button>
        ))}
      </div>
      <p className="text-faint text-xs">{t(`skill.${question.skill}`)}</p>
    </section>
  )
}

/** Done is the verdict screen: analyze a placement, hear a final's result. */
function Done({
  test,
  onClose,
  onAnalyzed,
}: {
  test: PathTest
  onClose: (analyzed: boolean) => void
  onAnalyzed: () => void
}) {
  const t = useTranslations("path.test")
  const analyze = useMutation({
    mutationFn: () => analyzeTest(test.id),
    onSuccess: () => {
      onAnalyzed()
      onClose(true)
    },
  })

  return (
    <div className="anim-fade mx-auto max-w-[560px] space-y-4 py-10 text-center lg:pt-16">
      <h1 className="text-2xl font-semibold tracking-[-0.01em]">
        {test.kind === "final"
          ? t(test.passed ? "finalPassed" : "finalFailed", {
              level: test.levelReached,
            })
          : t("placed", { level: test.levelReached })}
      </h1>
      <p className="text-muted-foreground text-sm leading-relaxed">
        {test.kind === "final"
          ? t(test.passed ? "finalPassedHint" : "finalFailedHint")
          : t("placedHint")}
      </p>
      {analyze.isError && (
        <p role="alert" className="text-destructive text-sm">
          {analyze.error.message}
        </p>
      )}
      <div className="flex justify-center gap-3">
        <Button variant="secondary" onClick={() => onClose(analyze.isSuccess)}>
          {t("exit")}
        </Button>
        {test.kind === "placement" && (
          <Button
            onClick={() => analyze.mutate()}
            size="lg"
            disabled={analyze.isPending}
            className="px-4 text-[13px]"
          >
            {t(analyze.isPending ? "analyzing" : "analyze")}
          </Button>
        )}
      </div>
    </div>
  )
}

