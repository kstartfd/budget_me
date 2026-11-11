"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronLeft } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useMutation } from "@tanstack/react-query"
import { analyzeTest, answerBatch } from "../api"
import { usePathTest } from "../hooks"
import type { PathTest, Preferences } from "../schemas"

/**
 * Runs one level test batch by batch: the ladder climbs while the batches
 * pass, and a finished placement hands over to the analysis that builds the
 * program.
 */
export function TestRunner({ id, onClose }: { id: string; onClose: (analyzed: boolean) => void }) {
  const t = useTranslations("path.test")
  const query = usePathTest(id)
  const [picked, setPicked] = useState<Record<string, string>>({})
  const [confirmed, setConfirmed] = useState<Record<string, boolean>>({})
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
      setConfirmed({})
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

  const answeredCount = current.questions.filter((q) => picked[q.key] && confirmed[q.key]).length
  const allAnswered = answeredCount === current.questions.length

  return (
    <div className="anim-fade space-y-4 lg:max-w-[704px]">
      <header className="space-y-1">
        <h1 className="text-[22px] font-semibold tracking-[-0.01em] lg:text-2xl">
          {t(current.kind === "final" ? "finalTitle" : "placementTitle", {
            level: current.level,
          })}
        </h1>
        <p className="text-muted-foreground text-[13px] lg:text-sm">
          {current.asked > 0
            ? t("progress", {
                correct: current.correct,
                asked: current.asked,
                needed: current.needed,
              })
            : t("progressFirst", { needed: current.needed })}
          {" · "}
          {t("answeredCount", { answered: answeredCount, total: current.questions.length })}
        </p>
      </header>

      <aside className="border-border bg-muted/40 text-muted-foreground rounded-lg border p-3.5 text-[13px] leading-relaxed">
        <p className="text-foreground mb-1 font-semibold">{t("howTitle")}</p>
        <ol className="list-decimal space-y-0.5 pl-5">
          <li>{t("howPick")}</li>
          <li>{t("howConfirm")}</li>
          <li>{t("howAccept")}</li>
        </ol>
      </aside>

      <div className="space-y-3">
        {current.questions.map((q, index) => (
          <QuestionCard
            key={q.key}
            index={index}
            question={q}
            picked={picked[q.key]}
            confirmed={!!confirmed[q.key]}
            onPick={(answer) => {
              setPicked((prev) => ({ ...prev, [q.key]: answer }))
              setConfirmed((prev) => ({ ...prev, [q.key]: false }))
            }}
            onConfirm={() => setConfirmed((prev) => ({ ...prev, [q.key]: true }))}
          />
        ))}
      </div>

      {submit.isError && (
        <p role="alert" className="text-destructive text-sm">
          {submit.error.message}
        </p>
      )}

      <p
        className={`text-center text-[13px] ${allAnswered ? "font-semibold" : "text-muted-foreground"}`}
        aria-live="polite"
      >
        {t("youAnswered", { answered: answeredCount, total: current.questions.length })}
      </p>

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
          {t(submit.isPending ? "sending" : "acceptAll")}
        </Button>
      </div>
    </div>
  )
}

function QuestionCard({
  index,
  question,
  picked,
  confirmed,
  onPick,
  onConfirm,
}: {
  index: number
  question: PathTest["questions"][number]
  picked: string | undefined
  confirmed: boolean
  onPick: (answer: string) => void
  onConfirm: () => void
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
        {picked && confirmed && (
          <span
            className="text-muted-foreground ml-2 text-xs font-normal"
            aria-label={t("answered")}
          >
            ✓ {t("answered")}
          </span>
        )}
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
                ? "bg-muted border-foreground font-semibold"
                : "bg-background"
            }`}
          >
            {option}
          </button>
        ))}
      </div>
      <div className="flex items-center justify-between gap-3">
        <p className="text-faint text-xs">{t(`skill.${question.skill}`)}</p>
        <Button
          type="button"
          size="sm"
          variant={confirmed ? "secondary" : "default"}
          disabled={!picked || confirmed}
          onClick={onConfirm}
        >
          {confirmed ? t("answered") : t("answer")}
        </Button>
      </div>
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
  const [prefs, setPrefs] = useState<Preferences>({})
  const analyze = useMutation({
    mutationFn: () => analyzeTest(test.id, prefs),
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
      {test.kind === "placement" && (
        <Survey value={prefs} onChange={setPrefs} disabled={analyze.isPending} />
      )}
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


const minuteOptions = [15, 30, 45, 60]
const dayOptions = [3, 5, 7]

/** Survey is the short questionnaire the program is built around. */
function Survey({
  value,
  onChange,
  disabled,
}: {
  value: Preferences
  onChange: (value: Preferences) => void
  disabled: boolean
}) {
  const t = useTranslations("path.survey")
  const set = (patch: Preferences) => onChange({ ...value, ...patch })

  const chips = (
    options: number[],
    current: number | undefined,
    pick: (n: number | undefined) => void,
    label: (n: number) => string,
  ) => (
    <div className="flex flex-wrap justify-center gap-2">
      {options.map((n) => (
        <Button
          key={n}
          type="button"
          size="sm"
          disabled={disabled}
          variant={current === n ? "default" : "outline"}
          aria-pressed={current === n}
          onClick={() => pick(current === n ? undefined : n)}
          className="h-auto rounded-full px-3 py-1 text-xs"
        >
          {label(n)}
        </Button>
      ))}
    </div>
  )

  return (
    <div className="border-border bg-background space-y-4 rounded-lg border p-4 text-left">
      <div>
        <p className="text-sm font-semibold">{t("title")}</p>
        <p className="text-muted-foreground text-xs leading-relaxed">
          {t("hint")}
        </p>
      </div>
      {(["interests", "goal", "variant"] as const).map((name) => (
        <label key={name} className="block space-y-1.5 text-[13px] font-medium">
          <span>{t(name)}</span>
          <Input
            value={value[name] ?? ""}
            maxLength={name === "variant" ? 60 : 200}
            disabled={disabled}
            placeholder={t(`${name}Placeholder`)}
            onChange={(e) => set({ [name]: e.target.value })}
            className="bg-background h-10 text-sm md:text-sm"
          />
        </label>
      ))}
      <div className="space-y-1.5 text-center">
        <p className="text-[13px] font-medium">{t("minutes")}</p>
        {chips(minuteOptions, value.minutesPerDay, (n) => set({ minutesPerDay: n }), (n) => t("minutesValue", { n }))}
      </div>
      <div className="space-y-1.5 text-center">
        <p className="text-[13px] font-medium">{t("days")}</p>
        {chips(dayOptions, value.daysPerWeek, (n) => set({ daysPerWeek: n }), (n) => t("daysValue", { n }))}
      </div>
    </div>
  )
}
