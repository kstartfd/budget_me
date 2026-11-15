"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import type { PathAttempt, PathExercise } from "../schemas"

/** Between the parts of a key and of an answer: gaps, replies. */
export const answerSep = " | "

/** The kinds that bring their own input: a sentence to put together, a text
 * with gaps, a text with questions. */
export const compoundKinds = ["order", "cloze", "read"] as const

export function isCompound(kind: PathExercise["kind"]): boolean {
  return (compoundKinds as readonly string[]).includes(kind)
}

const split = (answer: string) => answer.split("|").map((part) => part.trim())

const norm = (text: string) =>
  text.trim().toLowerCase().split(/\s+/).filter(Boolean).join(" ")

/** What the key says, for showing: one part to a line, and the other orders
 * of a sentence are left out. */
export function displayAnswer(exercise: PathExercise): string {
  if (exercise.kind === "order") return split(exercise.answer)[0]

  if (exercise.kind === "cloze" || exercise.kind === "read") {
    return split(exercise.answer)
      .map((part, i) => `${i + 1}. ${part.replaceAll("/", " / ")}`)
      .join("   ")
  }

  return exercise.answer
}

const bare = (text: string) =>
  text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim()

/** Whether a clue already holds the sentence of an "order" exercise: the
 * learner puts it together, nothing may show it before they ask. */
export function givesAway(text: string, answer: string): boolean {
  const haystack = ` ${bare(text)} `

  return split(answer).some((order) => bare(order) !== "" && haystack.includes(` ${bare(order)} `))
}

/** Whether one gap of a cloze holds what the key allows. */
function gapRight(key: string, given: string): boolean {
  return key.split("/").some((variant) => norm(variant) === norm(given))
}

type InputProps = {
  exercise: PathExercise
  /** The checked attempt, if there is one: it is what the verdict colours. */
  result?: PathAttempt
  pending: boolean
  onSubmit: (answer: string) => void
  revealed: boolean
  onReveal: () => void
}

function CheckRow({
  ready,
  pending,
  onReveal,
  revealed,
}: {
  ready: boolean
  pending: boolean
  onReveal: () => void
  revealed: boolean
}) {
  const t = useTranslations("path.topic")

  return (
    <div className="flex w-full gap-2 sm:w-auto">
      <Button
        type="submit"
        variant={ready ? "default" : "outline"}
        disabled={pending || !ready}
        className="h-10 flex-1 px-5 text-sm sm:flex-none"
      >
        {t(pending ? "checking" : "check")}
      </Button>
      <Button
        type="button"
        variant="ghost"
        onClick={onReveal}
        aria-expanded={revealed}
        className="text-muted-foreground h-10 px-3 text-sm underline underline-offset-4 hover:bg-transparent"
      >
        {t(revealed ? "hideAnswer" : "showAnswer")}
      </Button>
    </div>
  )
}

/**
 * The words of a sentence in a scrambled order: tap a word to put it into the
 * sentence, tap it there to take it back; on a computer a word can also be
 * dragged into the sentence, within it, and out of it.
 */
export function OrderInput({ exercise, result, pending, onSubmit, revealed, onReveal }: InputProps) {
  const t = useTranslations("path.topic")
  // Indices into exercise.options, in the order the learner put them.
  const [placed, setPlaced] = useState<number[]>([])
  const [dragged, setDragged] = useState<number | null>(null)
  const bank = exercise.options.map((_, i) => i).filter((i) => !placed.includes(i))
  const sentence = placed.map((i) => exercise.options[i]).join(" ")
  const failed = !!result && !result.correct && result.answer === sentence
  const done = !!result && result.correct && result.answer === sentence

  const put = (index: number, before?: number) => {
    setPlaced((current) => {
      const rest = current.filter((i) => i !== index)
      const at = before === undefined ? rest.length : rest.indexOf(before)

      return at < 0 ? [...rest, index] : [...rest.slice(0, at), index, ...rest.slice(at)]
    })
  }

  const take = (index: number) => setPlaced((current) => current.filter((i) => i !== index))

  const tile = (index: number, where: "bank" | "sentence") => (
    <button
      key={index}
      type="button"
      draggable
      onDragStart={(e) => {
        setDragged(index)
        e.dataTransfer.effectAllowed = "move"
        e.dataTransfer.setData("text/plain", exercise.options[index])
      }}
      onDragEnd={() => setDragged(null)}
      onDragOver={(e) => where === "sentence" && e.preventDefault()}
      onDrop={(e) => {
        if (where !== "sentence" || dragged === null) return
        e.preventDefault()
        e.stopPropagation()
        put(dragged, index)
        setDragged(null)
      }}
      onClick={() => (where === "bank" ? put(index) : take(index))}
      disabled={pending}
      className={cn(
        "border-input bg-background hover:bg-muted/60 min-h-9 cursor-grab touch-manipulation rounded-md border px-3 py-1.5 text-sm font-medium transition-colors select-none active:cursor-grabbing disabled:cursor-not-allowed",
        dragged === index && "opacity-40",
        where === "sentence" && done && "border-success bg-success/10 text-success",
        where === "sentence" && failed && "border-destructive bg-destructive/10 text-destructive",
      )}
    >
      {exercise.options[index]}
    </button>
  )

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (placed.length === exercise.options.length) onSubmit(sentence)
      }}
    >
      <div
        aria-label={t("orderSentence")}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          if (dragged !== null) put(dragged)
          setDragged(null)
        }}
        className={cn(
          "border-input flex min-h-14 flex-wrap items-center gap-2 rounded-lg border-2 border-dashed px-3 py-2.5",
          placed.length === 0 && "text-muted-foreground text-sm",
        )}
      >
        {placed.length === 0 ? t("orderEmpty") : placed.map((i) => tile(i, "sentence"))}
      </div>
      <div
        aria-label={t("orderWords")}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          if (dragged !== null) take(dragged)
          setDragged(null)
        }}
        className="flex min-h-12 flex-wrap gap-2"
      >
        {bank.map((i) => tile(i, "bank"))}
      </div>
      <div className="flex flex-wrap gap-2">
        <CheckRow
          ready={placed.length === exercise.options.length}
          pending={pending}
          onReveal={onReveal}
          revealed={revealed}
        />
        {placed.length > 0 && (
          <Button
            type="button"
            variant="ghost"
            onClick={() => setPlaced([])}
            className="text-muted-foreground h-10 px-3 text-sm hover:bg-transparent"
          >
            {t("orderReset")}
          </Button>
        )}
      </div>
    </form>
  )
}

/** A passage with a field in the place of every [n]. */
export function ClozeInput({ exercise, result, pending, onSubmit, revealed, onReveal }: InputProps) {
  const t = useTranslations("path.topic")
  const parts = exercise.text.split(/\[(\d{1,2})\]/)
  // parts alternate: text, gap number, text, gap number, … text.
  const gaps = Math.floor(parts.length / 2)
  const key = split(exercise.answer)
  const [values, setValues] = useState<string[]>(() => Array.from({ length: gaps }, () => ""))
  const filled = values.every((value) => value.trim() !== "")
  const joined = values.map((value) => value.trim()).join(answerSep)
  const checked = result?.answer === joined ? result : undefined

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (filled) onSubmit(joined)
      }}
    >
      <p className="text-[15px] leading-10">
        {parts.map((part, i) => {
          if (i % 2 === 0) return <span key={i}>{part}</span>

          const n = (i - 1) / 2
          const right = checked ? gapRight(key[n] ?? "", values[n]) : undefined

          return (
            <span key={i} className="mx-1 inline-flex items-center align-baseline">
              <span className="text-muted-foreground mr-1 text-xs tabular-nums">{n + 1}</span>
              <Input
                value={values[n]}
                disabled={pending}
                onChange={(e) =>
                  setValues((current) => current.map((value, j) => (j === n ? e.target.value : value)))
                }
                aria-label={t("gap", { n: n + 1 })}
                aria-invalid={right === false}
                autoComplete="off"
                autoCapitalize="none"
                spellCheck={false}
                className={cn(
                  "bg-background h-9 w-32 text-sm md:text-sm",
                  right === true && "border-success bg-success/10 text-success",
                  right === false && "border-destructive bg-destructive/10 text-destructive",
                )}
              />
            </span>
          )
        })}
      </p>
      <div className="flex flex-wrap gap-2">
        <CheckRow ready={filled} pending={pending} onReveal={onReveal} revealed={revealed} />
      </div>
    </form>
  )
}

/** Questions about a passage; a question that ends in an ellipsis is a
 * sentence to finish, the others are answered in a word or a phrase. */
export function ReadInput({ exercise, pending, onSubmit, revealed, onReveal }: InputProps) {
  const t = useTranslations("path.topic")
  const [values, setValues] = useState<string[]>(() => exercise.options.map(() => ""))
  const filled = values.every((value) => value.trim() !== "")

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (filled) onSubmit(values.map((value) => value.trim()).join(answerSep))
      }}
    >
      <ol className="space-y-3">
        {exercise.options.map((question, i) => {
          const completion = /(\.{3}|…)\s*$/.test(question)
          const stem = question.replace(/\s*(\.{3}|…)\s*$/, "")

          return (
            <li key={i} className="space-y-1.5">
              <label className="text-[15px] leading-snug font-medium" htmlFor={`q-${exercise.id}-${i}`}>
                <span className="text-muted-foreground mr-1.5 text-xs tabular-nums">{i + 1}.</span>
                {stem}
                {completion && <span className="text-muted-foreground"> …</span>}
              </label>
              <Input
                id={`q-${exercise.id}-${i}`}
                value={values[i]}
                disabled={pending}
                onChange={(e) =>
                  setValues((current) => current.map((value, j) => (j === i ? e.target.value : value)))
                }
                placeholder={t(completion ? "readComplete" : "readAnswer")}
                autoComplete="off"
                className="bg-background h-10 text-sm md:text-sm"
              />
            </li>
          )
        })}
      </ol>
      <div className="flex flex-wrap gap-2">
        <CheckRow ready={filled} pending={pending} onReveal={onReveal} revealed={revealed} />
      </div>
    </form>
  )
}

/** The input of one of the kinds above. */
export function CompoundInput(props: InputProps) {
  if (props.exercise.kind === "order") return <OrderInput {...props} />
  if (props.exercise.kind === "cloze") return <ClozeInput {...props} />

  return <ReadInput {...props} />
}
