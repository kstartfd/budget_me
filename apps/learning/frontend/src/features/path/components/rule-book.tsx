"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ApiError } from "../api"
import { useRule, useRules } from "../hooks"
import { rulesButtonAnchor, useReturnPoint } from "../return-point"
import { ExplainSwitch } from "./explain-switch"
import { RuleCard, rulePath, rulesPath } from "./rule-card"
import { RulePractice } from "./rule-practice"

const wrap = "anim-fade space-y-4 lg:mx-auto lg:w-full lg:max-w-[704px]"

/** Said where the rule book is closed: it opens for a teacher whose language it covers. */
function Closed() {
  const t = useTranslations("path")

  return (
    <div role="alert" className="border-border space-y-3 rounded-lg border p-5">
      <p className="text-sm leading-relaxed">{t("rulesClosed")}</p>
      <Button asChild variant="secondary" size="lg" className="text-[13px]">
        <Link href="/learning/path">{t("rulesToTeachers")}</Link>
      </Button>
    </div>
  )
}

/**
 * RulesIndex is the rule book: every rule of the learner's teacher, by level,
 * with a search. A rule opens on a page of its own.
 */
export function RulesIndex({ language }: { language?: string }) {
  const t = useTranslations("path")
  const rules = useRules(language)
  const [query, setQuery] = useState("")
  // The spot of the teacher's screen the learner came from, if they came from it.
  const from = useReturnPoint()

  if (rules.isError)
    return (
      <div className={wrap}>
        {rules.error instanceof ApiError && rules.error.status === 403 ? (
          <Closed />
        ) : (
          <p role="alert" className="text-destructive text-sm">
            {rules.error.message}
          </p>
        )}
      </div>
    )

  const needle = query.trim().toLowerCase()
  const shown = (rules.data ?? []).filter(
    (rule) => !needle || `${rule.title} ${rule.unit}`.toLowerCase().includes(needle),
  )
  const levels = Array.from(new Set(shown.map((rule) => rule.level)))

  return (
    <div className={wrap}>
      <header className="space-y-1">
        <Button asChild variant="ghost" size="sm" className="-ml-2 h-auto px-2 py-1 text-xs">
          <Link href={from?.href ?? "/learning/path"}>
            <IconChevronLeft size={14} />
            {t(from ? "rulesBackToProgram" : "backToTeachers")}
          </Link>
        </Button>
        <h1 className="text-[22px] font-semibold tracking-[-0.01em] lg:text-2xl">
          {t("rulesTitle")}
        </h1>
        <p className="text-muted-foreground text-xs leading-relaxed lg:text-sm">
          {t("rulesSubtitle")}
        </p>
      </header>

      <ExplainSwitch language={language} />

      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t("rulesSearch")}
        aria-label={t("rulesSearch")}
        className="bg-background h-10 text-sm md:text-sm"
      />

      {rules.isPending && (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
        </div>
      )}

      {rules.isSuccess && shown.length === 0 && (
        <p className="text-muted-foreground text-sm">{t("rulesEmpty")}</p>
      )}

      {levels.map((level) => (
        <section key={level} className="space-y-2">
          <h2 className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
            {level}
          </h2>
          <ul className="space-y-1.5">
            {shown
              .filter((rule) => rule.level === level)
              .map((rule) => (
                <li key={rule.id}>
                  <Link
                    href={rulePath(rule.id, language)}
                    className="bg-background border-border hover:bg-muted/50 flex items-center gap-3 rounded-lg border px-4 py-3 transition-colors"
                  >
                    <span className="min-w-0 flex-1 text-sm leading-snug font-medium">
                      {rule.title}
                    </span>
                    <IconChevronRight size={14} className="text-faint shrink-0" aria-hidden="true" />
                  </Link>
                </li>
              ))}
          </ul>
        </section>
      ))}
    </div>
  )
}

/** One rule on a page of its own, reached from the book or from another rule. */
export function RulePage({ id, language }: { id: string; language?: string }) {
  const t = useTranslations("path")
  const rule = useRule(id, language)
  // A rule opened straight from a topic goes back to that topic, not to the book.
  const from = useReturnPoint()
  const fromTopic = !!from && from.anchor !== rulesButtonAnchor

  return (
    <div className={wrap}>
      <Button asChild variant="ghost" size="sm" className="-ml-2 h-auto px-2 py-1 text-xs">
        <Link href={fromTopic ? from.href : rulesPath(language)}>
          <IconChevronLeft size={14} />
          {t(fromTopic ? "rulesBackToTopic" : "rulesBack")}
        </Link>
      </Button>

      {rule.isPending && (
        <div className="space-y-3">
          <Skeleton className="h-8 w-2/3 rounded-lg" />
          <Skeleton className="h-32 w-full rounded-lg" />
        </div>
      )}

      {rule.isError &&
        (rule.error instanceof ApiError && rule.error.status === 403 ? (
          <Closed />
        ) : (
          <p role="alert" className="text-destructive text-sm">
            {rule.error.message}
          </p>
        ))}

      {rule.isSuccess && (
        <>
          <div className="bg-background border-border rounded-lg border p-5">
            <RuleCard rule={rule.data} language={language} withTitle />
          </div>
          <RulePractice ruleId={id} />
        </>
      )}
    </div>
  )
}
