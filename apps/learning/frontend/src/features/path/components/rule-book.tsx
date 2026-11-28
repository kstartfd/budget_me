"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconChevronLeft, IconChevronRight, IconSearch } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ApiError } from "../api"
import { useRule, useRules } from "../hooks"
import { rulesButtonAnchor, useReturnPoint } from "../return-point"
import { ExplainSwitch } from "./explain-switch"
import { RuleCard, rulePath, rulesPath } from "./rule-card"
import { RulePractice } from "./rule-practice"

const wrap = "anim-fade space-y-4 lg:mx-auto lg:w-full lg:max-w-[1120px]"

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
  const workspace = useTranslations("workspace")
  const rules = useRules(language)
  const [query, setQuery] = useState("")
  const [category, setCategory] = useState("")
  const [level, setLevel] = useState("")
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

  const all = rules.data ?? []
  const needle = query.trim().toLocaleLowerCase()
  const categories = Array.from(new Set(all.map((rule) => rule.unit))).sort((a, b) => a.localeCompare(b))
  const levels = Array.from(new Set(all.map((rule) => rule.level))).sort((a, b) => a.localeCompare(b))
  const shown = all.filter((rule) =>
    (!category || rule.unit === category) &&
    (!level || rule.level === level) &&
    (!needle || `${rule.title} ${rule.unit} ${rule.level}`.toLocaleLowerCase().includes(needle)),
  )

  return (
    <div className={wrap}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <header className="space-y-1">
          <h2 className="text-lg font-semibold tracking-tight">{t("rulesTitle")}</h2>
          <p className="text-muted-foreground text-xs leading-relaxed lg:text-sm">{t("rulesSubtitle")}</p>
        </header>
        {from && <Button asChild variant="outline" size="sm"><Link href={from.href}><IconChevronLeft size={14} />{t("rulesBackToProgram")}</Link></Button>}
      </div>

      <ExplainSwitch language={language} />

      <div className="border-border bg-card flex items-center gap-2 rounded-lg border px-3">
        <IconSearch size={17} className="text-muted-foreground" aria-hidden="true" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("rulesSearch")}
          aria-label={t("rulesSearch")}
          className="h-11 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0 md:text-sm"
        />
      </div>

      {rules.isPending && (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
        </div>
      )}

      {rules.isSuccess && <div className="grid items-start gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
        <aside className="border-border bg-card rounded-xl border p-3" aria-label={workspace("categories") }>
          <p className="text-muted-foreground mb-2 px-2 text-[11px] font-semibold tracking-wide uppercase">{workspace("categories")}</p>
          <button type="button" onClick={() => setCategory("")} aria-pressed={!category} className={`flex w-full justify-between rounded-md px-2.5 py-2 text-left text-sm ${!category ? "bg-muted font-semibold" : "hover:bg-muted/60"}`}>
            {workspace("allRules")} <span>{all.length}</span>
          </button>
          {categories.map((unit) => <button key={unit} type="button" onClick={() => setCategory(unit)} aria-pressed={category === unit} className={`flex w-full justify-between gap-2 rounded-md px-2.5 py-2 text-left text-sm ${category === unit ? "bg-muted font-semibold" : "hover:bg-muted/60"}`}>
            <span>{unit}</span><span className="text-muted-foreground">{all.filter((rule) => rule.unit === unit).length}</span>
          </button>)}
        </aside>
        <div className="min-w-0 space-y-3">
          <section className="border-border bg-card rounded-xl border p-5">
            <p className="text-muted-foreground text-xs">{workspace("referenceLibrary")}</p>
            <h3 className="mt-1 text-lg font-semibold">{workspace("grammarRules")}</h3>
            <p className="text-muted-foreground mt-1 text-sm">{workspace("ruleCount", { count: all.length })}</p>
            <p className="text-muted-foreground mt-2 text-xs">{workspace("ruleLevelNote")}</p>
          </section>
          <div className="flex items-center justify-between gap-3">
            <p className="text-muted-foreground text-xs">{workspace("filteredCount", { count: shown.length })}</p>
            <label className="text-muted-foreground flex items-center gap-2 text-xs">{workspace("levelFilter")}
              <select value={level} onChange={(event) => setLevel(event.target.value)} className="border-border bg-card min-h-9 rounded-md border px-2 text-sm">
                <option value="">{workspace("allLevels")}</option>
                {levels.map((value) => <option key={value} value={value}>{value}</option>)}
              </select>
            </label>
          </div>
          {shown.length === 0 ? <p className="border-border bg-card rounded-xl border p-5 text-sm text-muted-foreground">{t("rulesEmpty")}</p> :
            <ul className="border-border bg-card overflow-hidden rounded-xl border">
              {shown.map((rule) => <li key={rule.id} className="border-border border-t first:border-t-0">
                <Link href={rulePath(rule.id, language)} className="hover:bg-muted/50 focus-visible:bg-muted/50 flex min-h-20 items-center gap-4 px-4 py-3 transition-colors focus-visible:outline-2 focus-visible:outline-ring">
                  <span className="min-w-0 flex-1"><span className="block text-sm font-medium">{rule.title} <span className="text-muted-foreground text-xs font-normal">{rule.level}</span></span><span className="text-muted-foreground mt-1 block text-xs">{rule.unit}</span></span>
                  <span className="border-border rounded-md border px-2.5 py-1.5 text-xs">{workspace("previewRule")}</span><IconChevronRight size={14} className="text-muted-foreground" aria-hidden="true" />
                </Link>
              </li>)}
            </ul>}
        </div>
      </div>}
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
