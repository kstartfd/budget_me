"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconCheck, IconChevronRight, IconX } from "@tabler/icons-react"
import { cn } from "@/lib/utils"
import type { PathRule } from "../schemas"

/** The path of a rule's own page. */
export const rulePath = (id: string) => `/learning/path/rules/${id}`

function Heading({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-muted-foreground text-[10px] font-semibold tracking-wider uppercase">
      {children}
    </h3>
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
  withTitle = false,
  className,
}: {
  rule: PathRule
  /** The page of a rule has its own title above; inside a topic it is shown here. */
  withTitle?: boolean
  className?: string
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
        <p className="text-[14px] leading-relaxed">{rule.theory}</p>
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
                <p className="text-[14px] font-medium">{example.text}</p>
                <p className="text-muted-foreground text-[13px]">{example.translation}</p>
              </li>
            ))}
          </ul>
        </section>
      )}

      {rule.mistakes.length > 0 && (
        <section className="space-y-1.5">
          <Heading>{t("ruleMistakes")}</Heading>
          <ul className="space-y-2">
            {rule.mistakes.map((mistake) => (
              <li key={mistake.wrong} className="space-y-1 text-[13px]">
                <p className="text-destructive flex items-start gap-1.5">
                  <IconX size={14} className="mt-0.5 shrink-0" aria-hidden="true" />
                  <span className="line-through decoration-1">{mistake.wrong}</span>
                </p>
                <p className="text-success flex items-start gap-1.5 font-medium">
                  <IconCheck size={14} className="mt-0.5 shrink-0" aria-hidden="true" />
                  <span>{mistake.right}</span>
                </p>
                {mistake.note && (
                  <p className="text-muted-foreground pl-5 leading-snug">{mistake.note}</p>
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
                  href={rulePath(other.id)}
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
