"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconCheck, IconChevronRight, IconX } from "@tabler/icons-react"
import { cn } from "@/lib/utils"
import { exampleLineClass, withExamples } from "@/components/example"
import { seeAlsoAnchor } from "../return-point"
import type { PathRule } from "../schemas"

/** The path of a rule's own page, in the rule book of the teacher of a language. */
export const rulePath = (id: string, language?: string) =>
  `/learning/path/rules/${id}${bookQuery(language)}`

/** The path of the rule book of the teacher of a language. */
export const rulesPath = (language?: string) =>
  `/learning/path/rules${bookQuery(language)}`

const bookQuery = (language?: string) =>
  language ? `?language=${encodeURIComponent(language)}` : ""

/** A label of a section: small caps, but big enough to read (12px) and in the muted text colour. */
export function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
      {children}
    </h3>
  )
}

const Heading = SectionLabel

/** A run of Latin letters, with the spaces, commas and plus signs that join its words. */
const english = /([A-Za-z][A-Za-z0-9'’/\-]*(?:[ ,+]+[A-Za-z0-9][A-Za-z0-9'’/\-]*)*[.?!]?)/g

/** A letter of a script other than Latin: the text is written in the learner's own language. */
const otherScript = /(?!\p{Script=Latin})\p{L}/u

/**
 * The look of the learned language inside an explanation: the readability
 * sheet's blue wash, 8:1 on the card in both themes, in place of a heavier
 * weight that was hard to tell from the text around it.
 */
export const englishClass =
  "bg-lang text-lang-foreground rounded-[4px] px-[3px] py-px font-medium [box-decoration-break:clone]"

/**
 * Sets the examples of a text (between backticks) in their own font, and in the
 * rest of it, when it is written in another script than Latin, picks out the
 * English. A text with no letters of another script is not an explanation of
 * English at all (the learner's own language is written in Latin too), so
 * there is nothing more to pick out and it stays as it is.
 */
function withMarks(text: string) {
  const mixed = otherScript.test(text)

  return withExamples(text, (part) => (mixed ? withEnglish(part) : part))
}

/** The English in a part of an explanation written in the learner's language, on a blue wash. */
function withEnglish(text: string) {
  return text.split(english).map((part, index) => {
    if (index % 2 === 0) return part

    // The full stop that ends the learner's own sentence is not part of a short phrase.
    const stop = /[.?!]$/.test(part) && part.trim().split(/\s+/).length < 4 ? part.slice(-1) : ""

    return (
      <span key={index}>
        <span className={englishClass}>{stop ? part.slice(0, -1) : part}</span>
        {stop}
      </span>
    )
  })
}

/** An explanation in the learner's language, the English in it picked out; one line of text, no paragraphs. */
export function Mixed({ text }: { text: string }) {
  return <>{withMarks(text)}</>
}

/** The share of a sentence's letters that are Latin: an example in English is nearly all of them. */
const latinShare = (sentence: string) => {
  const letters = sentence.match(/\p{L}/gu)?.length ?? 0
  if (letters === 0) return 0

  return (sentence.match(/[A-Za-z]/g)?.length ?? 0) / letters
}

/**
 * Cuts an explanation into paragraphs: a sentence opens a new one, and the
 * examples that follow it stay with it. An example is a sentence set between
 * backticks; in a text of the learner's own language, English sentences with no
 * mark count as examples too (older texts have none). The library writes a rule
 * as one run of text, which read as a wall.
 */
export function paragraphsOf(text: string): string[] {
  const sentences = text.split(/(?<=[.!?]`?)\s+(?=`?[A-ZА-ЯЁ])/u)
  const mixed = otherScript.test(text)
  const out: string[] = []

  for (const sentence of sentences) {
    const example = sentence.startsWith("`") || (mixed && latinShare(sentence) > 0.8)

    if (out.length > 0 && example) out[out.length - 1] += ` ${sentence}`
    else out.push(sentence)
  }

  return out
}

/**
 * Prose is an explanation in the learner's language that holds English: cut
 * into short paragraphs, a line no wider than it reads comfortably, the
 * English picked out on a blue wash.
 */
export function Prose({ text, className }: { text: string; className?: string }) {
  return (
    <div className={cn("max-w-[65ch] space-y-2.5 text-[15px] leading-[1.6]", className)}>
      {paragraphsOf(text).map((paragraph) => (
        <p key={paragraph}>{withMarks(paragraph)}</p>
      ))}
    </div>
  )
}

/**
 * RuleCard is one rule of the rule book: how it is built, what it says,
 * sentences that show it, the mistakes made against it and the rules it is read
 * together with. The references lead to those rules' own pages. Inside a topic
 * it comes first, before the exercises; on the rule's page it is the page.
 */
export function RuleCard({
  rule,
  language,
  withTitle = false,
  className,
  onLeave,
}: {
  rule: PathRule
  /** The language of the teacher whose book this is: the rules it refers to are of the same book. */
  language?: string
  /** The page of a rule has its own title above; inside a topic it is shown here. */
  withTitle?: boolean
  className?: string
  /** Called with the rule whose link is pressed, so a screen can mark where it was left. */
  onLeave?: (ruleId: string) => void
}) {
  const t = useTranslations("path")

  return (
    <article className={cn("space-y-4", className)}>
      {withTitle && (
        <header className="space-y-1">
          <p className="text-muted-foreground text-xs font-semibold">
            {rule.level} · {rule.unit}
          </p>
          <h2 className="text-lg leading-snug font-semibold">{rule.title}</h2>
        </header>
      )}

      <section className="space-y-1.5">
        <Heading>{t("ruleTheory")}</Heading>
        <Prose text={rule.theory} />
      </section>

      {rule.forms.length > 0 && (
        <section className="space-y-1.5">
          <Heading>{t("ruleForms")}</Heading>
          <ul className="space-y-1.5">
            {rule.forms.map((form) => (
              <li
                key={form}
                className="bg-muted rounded-md px-3 py-2 text-[13px] leading-snug"
              >
                {form}
              </li>
            ))}
          </ul>
        </section>
      )}

      {rule.examples.length > 0 && (
        <section className="space-y-1.5">
          <Heading>{t("ruleExamples")}</Heading>
          <ul className="space-y-2">
            {rule.examples.map((example) => (
              <li key={example.text} className="border-border border-l-2 pl-3">
                <p className={cn("text-[14px] leading-snug font-medium", exampleLineClass)}>{example.text}</p>
                <p className="text-muted-foreground text-[13px] leading-snug">{example.translation}</p>
              </li>
            ))}
          </ul>
        </section>
      )}

      {rule.mistakes.length > 0 && (
        <section className="space-y-1.5">
          <Heading>{t("ruleMistakes")}</Heading>
          <ul className="space-y-3">
            {rule.mistakes.map((mistake) => (
              <li key={mistake.wrong} className="space-y-1.5">
                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="border-destructive/30 bg-destructive/5 space-y-1 rounded-lg border px-3 py-2.5">
                    <p className="text-destructive flex items-center gap-1 text-xs font-semibold">
                      <IconX size={14} aria-hidden="true" />
                      {t("ruleWrong")}
                    </p>
                    <p className={cn("text-[13px] leading-snug", exampleLineClass)}>{mistake.wrong}</p>
                  </div>
                  <div className="border-success/30 bg-success/5 space-y-1 rounded-lg border px-3 py-2.5">
                    <p className="text-success flex items-center gap-1 text-xs font-semibold">
                      <IconCheck size={14} aria-hidden="true" />
                      {t("ruleRight")}
                    </p>
                    <p className={cn("text-[13px] leading-snug font-medium", exampleLineClass)}>{mistake.right}</p>
                  </div>
                </div>
                {mistake.note && (
                  <p className="text-muted-foreground max-w-[65ch] text-[13px] leading-snug">
                    <Mixed text={mistake.note} />
                  </p>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      {rule.related.length > 0 && (
        <section className="space-y-1.5">
          <Heading>{t("ruleSeeAlso")}</Heading>
          <ul className="space-y-1.5">
            {rule.related.map((other) => (
              <li key={other.id}>
                <Link
                  href={rulePath(other.id, language)}
                  data-return-anchor={seeAlsoAnchor(other.id)}
                  onClick={() => onLeave?.(other.id)}
                  className="border-border hover:bg-muted/50 flex items-center gap-2 rounded-md border px-3 py-2 text-[13px] transition-colors"
                >
                  <span className="bg-secondary rounded-full px-2 py-0.5 text-[11px] font-semibold">
                    {other.level}
                  </span>
                  <span className="min-w-0 flex-1 leading-snug">{other.title}</span>
                  <IconChevronRight size={14} className="text-faint shrink-0" aria-hidden="true" />
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </article>
  )
}
