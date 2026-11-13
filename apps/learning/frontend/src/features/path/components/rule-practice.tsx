"use client"

import { useEffect, useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import {
  IconCheck,
  IconMessageCircle,
  IconSparkles,
  IconX,
} from "@tabler/icons-react"
import { exampleLineClass } from "@/components/example"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/features/auth"
import { ExampleChat } from "@/features/chat/components/example-chat"
import { cn } from "@/lib/utils"
import { ApiError, checkPracticeItem, fetchPracticeKey, leavePractice } from "../api"
import {
  practiceKey,
  usePractice,
  useWriteExamples,
  useWriteExercises,
} from "../hooks"
import type { PracticeBlock, PracticeExample, PracticeItem } from "../schemas"
import { Mixed } from "./rule-card"

/** The most sentences a sitting holds: the same cap the server keeps. */
const maxExamples = 20

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="text-sm font-semibold">{children}</h3>
}

/**
 * RulePractice is what the learner's agent adds to a rule: more example
 * sentences, each one open to a chat, and drills in the manner of English
 * Grammar in Use. None of it is kept: it lives while the learner is on the
 * rule's page and goes when they leave, which the server is told.
 */
export function RulePractice({ ruleId }: { ruleId: string }) {
  const t = useTranslations("path.practice")
  const practice = usePractice(ruleId)
  const examples = useWriteExamples(ruleId)
  const exercises = useWriteExercises(ruleId)
  // New drills start from a clean sheet: no verdict of the old set is kept.
  const [round, setRound] = useState(0)

  const list = practice.data?.examples ?? []
  const blocks = practice.data?.blocks ?? []
  const sitting = list.length + blocks.length > 0

  // Leaving the page (a route change, a closed tab, a closed mini app) tells the
  // server, which then drops the sitting after a short grace.
  const live = useRef(false)

  useEffect(() => {
    live.current = sitting
  }, [sitting])

  useEffect(() => {
    const leave = () => {
      if (live.current) leavePractice(ruleId)
    }

    window.addEventListener("pagehide", leave)

    return () => {
      window.removeEventListener("pagehide", leave)
      leave()
    }
  }, [ruleId])

  return (
    <section className="bg-background border-border space-y-6 rounded-lg border p-5">
      <header className="space-y-1">
        <h2 className="text-lg font-semibold">{t("title")}</h2>
        <p className="text-muted-foreground text-[13px] leading-relaxed">{t("note")}</p>
      </header>

      {practice.isError && (
        <p role="alert" className="text-destructive text-sm">
          {practice.error.message}
        </p>
      )}

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <SectionTitle>{t("examplesTitle")}</SectionTitle>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => examples.mutate()}
            disabled={examples.isPending || practice.isPending || list.length >= maxExamples}
            className="gap-1.5 text-[13px]"
          >
            <IconSparkles size={15} aria-hidden="true" />
            {examples.isPending ? t("writing") : t(list.length > 0 ? "examplesMore" : "examplesMake")}
          </Button>
        </div>

        {examples.isError && (
          <p role="alert" className="text-destructive text-sm">
            {examples.error.message}
          </p>
        )}

        {practice.isPending && <Skeleton className="h-10 w-full rounded-lg" />}

        {practice.isSuccess && list.length === 0 && (
          <p className="text-muted-foreground text-[13px] leading-relaxed">{t("examplesEmpty")}</p>
        )}

        {list.length > 0 && (
          <ul className="space-y-4">
            {list.map((example) => (
              <Example key={example.id} ruleId={ruleId} example={example} />
            ))}
          </ul>
        )}
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <SectionTitle>{t("exercisesTitle")}</SectionTitle>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => exercises.mutate(undefined, { onSuccess: () => setRound((r) => r + 1) })}
            disabled={exercises.isPending || practice.isPending}
            className="gap-1.5 text-[13px]"
          >
            <IconSparkles size={15} aria-hidden="true" />
            {exercises.isPending ? t("writing") : t(blocks.length > 0 ? "exercisesNew" : "exercisesMake")}
          </Button>
        </div>

        {exercises.isError && (
          <p role="alert" className="text-destructive text-sm">
            {exercises.error.message}
          </p>
        )}

        {practice.isSuccess && blocks.length === 0 && (
          <p className="text-muted-foreground text-[13px] leading-relaxed">{t("exercisesEmpty")}</p>
        )}

        <div className="space-y-5">
          {blocks.map((block, i) => (
            <Block key={`${round}-${block.id}`} ruleId={ruleId} block={block} number={i + 1} />
          ))}
        </div>
      </div>
    </section>
  )
}

/** One sentence the agent wrote, with its translation and a chat about it. */
function Example({ ruleId, example }: { ruleId: string; example: PracticeExample }) {
  const t = useTranslations("path.practice")
  const [open, setOpen] = useState(false)

  return (
    <li className="border-border space-y-2 border-l-2 pl-3">
      <div>
        <p className={cn("text-[14px] font-medium", exampleLineClass)}>{example.text}</p>
        <p className="text-muted-foreground text-[13px]">{example.translation}</p>
      </div>
      <Button
        variant="ghost"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="text-primary h-auto gap-1.5 px-0 py-0 text-[13px] font-medium"
      >
        <IconMessageCircle size={15} aria-hidden="true" />
        {t(open ? "discussHide" : "discuss")}
      </Button>
      {open && <ExampleChat ruleId={ruleId} exampleId={example.id} />}
    </li>
  )
}

/** What the learner was told about one line of a drill. */
interface Verdict {
  status: "right" | "wrong" | "shown"
  key: string
  note: string
  answer?: string
}

/** One drill: an instruction and its lines, with a score once every line is answered. */
function Block({
  ruleId,
  block,
  number,
}: {
  ruleId: string
  block: PracticeBlock
  number: number
}) {
  const t = useTranslations("path.practice")
  const [verdicts, setVerdicts] = useState<Record<string, Verdict>>({})

  const answered = block.items.filter((item) => verdicts[item.id]).length
  const right = block.items.filter((item) => verdicts[item.id]?.status === "right").length

  return (
    <section className="border-border space-y-4 border-t pt-5 first:border-t-0 first:pt-0">
      <header className="flex items-start gap-3">
        <span className="bg-foreground text-background grid size-6 shrink-0 place-items-center rounded-full text-xs font-semibold">
          {number}
        </span>
        <p className="text-[14px] leading-snug font-medium">
          <Mixed text={block.instruction} />
        </p>
      </header>

      <ol className="divide-border divide-y">
        {block.items.map((item, i) => (
          <Line
            key={item.id}
            ruleId={ruleId}
            block={block}
            item={item}
            index={i}
            verdict={verdicts[item.id]}
            onVerdict={(v) => setVerdicts((all) => ({ ...all, [item.id]: v }))}
          />
        ))}
      </ol>

      {answered === block.items.length && (
        <p className="text-sm font-semibold">{t("score", { right, total: block.items.length })}</p>
      )}
    </section>
  )
}

/** One line of a drill: the sentence, how to answer it and what the check said. */
function Line({
  ruleId,
  block,
  item,
  index,
  verdict,
  onVerdict,
}: {
  ruleId: string
  block: PracticeBlock
  item: PracticeItem
  index: number
  verdict: Verdict | undefined
  onVerdict: (verdict: Verdict) => void
}) {
  const t = useTranslations("path.practice")
  const client = useQueryClient()
  const { user } = useAuth()
  const [value, setValue] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")

  // null asks for the key; a string is an answer to be graded.
  const run = async (answer: string | null) => {
    if (busy) return

    setBusy(true)
    setError("")

    try {
      if (answer === null) {
        const key = await fetchPracticeKey(ruleId, block.id, item.id)
        onVerdict({ status: "shown", key: key.key, note: key.note })
      } else {
        const check = await checkPracticeItem(ruleId, block.id, item.id, answer)
        onVerdict({ status: check.correct ? "right" : "wrong", key: check.key, note: check.note, answer })
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))

      // The sitting is gone (its time ran out): the page shows what is left.
      if (e instanceof ApiError && e.status === 404) {
        void client.invalidateQueries({ queryKey: practiceKey(user?.id, ruleId) })
      }
    } finally {
      setBusy(false)
    }
  }

  const gapped = block.kind === "form" || block.kind === "choose"
  const [before, after = ""] = gapped ? item.text.split("...") : [item.text]
  // The word in brackets is shown once, whether or not the agent already put it in the sentence.
  const showHint = block.kind === "form" && !!item.hint && !item.text.includes(`(${item.hint})`)
  // What stands in the gap once the line is settled: the learner's answer, or the key they asked for.
  const filled = verdict && (verdict.status === "shown" ? verdict.key : verdict.answer)

  return (
    <li className="space-y-2.5 py-4 first:pt-0 last:pb-0">
      <p className="text-[15px] leading-relaxed">
        <span className="text-muted-foreground mr-1.5 text-[13px] tabular-nums">{index + 1}.</span>
        {before}
        {gapped && (
          <span
            className={cn(
              "border-foreground/40 mx-1 inline-block min-w-14 border-b text-center font-medium",
              verdict?.status === "right" && "border-success text-success",
              verdict?.status === "wrong" && "border-destructive text-destructive",
            )}
          >
            {filled ?? " "}
          </span>
        )}
        {after}
        {showHint && <span className="text-muted-foreground"> ({item.hint})</span>}
      </p>

      {block.kind === "make" && item.hint && (
        <p className="text-muted-foreground text-[13px]">→ {item.hint}</p>
      )}

      {block.kind === "choose" ? (
        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
          {item.options.map((option) => {
            const isKey = verdict && option === verdict.key
            const isMiss = verdict?.status === "wrong" && option === verdict.answer

            return (
              <Button
                key={option}
                type="button"
                variant="outline"
                disabled={!!verdict || busy}
                onClick={() => void run(option)}
                className={cn(
                  "h-10 text-sm",
                  isKey && "border-success text-success disabled:opacity-100",
                  isMiss && "border-destructive text-destructive disabled:opacity-100",
                )}
              >
                {option}
              </Button>
            )
          })}
        </div>
      ) : (
        !verdict && (
          <form
            className="flex flex-col gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              if (value.trim()) void run(value)
            }}
          >
            <Input
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={t(block.kind === "form" ? "placeholderWord" : "placeholderSentence")}
              aria-label={item.text}
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              className="bg-background h-10 text-sm md:text-sm"
            />
            <div className="flex items-center gap-4">
              <Button
                type="submit"
                variant={value.trim() ? "default" : "outline"}
                disabled={busy || !value.trim()}
                className="h-10 flex-1 px-5 text-sm sm:flex-none"
              >
                {t("check")}
              </Button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void run(null)}
                className="text-muted-foreground hover:text-foreground shrink-0 text-[13px] underline underline-offset-2 disabled:opacity-50"
              >
                {t("showAnswer")}
              </button>
            </div>
          </form>
        )
      )}

      {block.kind === "choose" && !verdict && (
        <button
          type="button"
          disabled={busy}
          onClick={() => void run(null)}
          className="text-muted-foreground hover:text-foreground shrink-0 text-[13px] underline underline-offset-2 disabled:opacity-50"
        >
          {t("showAnswer")}
        </button>
      )}

      {error && (
        <p role="alert" className="text-destructive text-[13px]">
          {error}
        </p>
      )}

      {verdict && (
        <div className="space-y-1 text-[13px] leading-snug">
          {verdict.status === "right" && (
            <p className="text-success flex items-center gap-1.5 font-medium">
              <IconCheck size={14} aria-hidden="true" />
              {t("right")}
            </p>
          )}
          {verdict.status === "wrong" && (
            <p className="text-destructive flex items-center gap-1.5 font-medium">
              <IconX size={14} aria-hidden="true" />
              {t("wrong")}
            </p>
          )}
          {verdict.status !== "right" && (
            <p>
              <span className="text-muted-foreground">{t("keyIs")}: </span>
              <strong>{verdict.key}</strong>
            </p>
          )}
          {verdict.note && (
            <p className="text-muted-foreground">
              <Mixed text={verdict.note} />
            </p>
          )}
        </div>
      )}
    </li>
  )
}
