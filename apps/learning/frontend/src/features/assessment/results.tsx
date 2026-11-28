"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconCheck, IconChevronDown } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { card, Pill, type Tone } from "@/features/home/components/parts"
import { skillIds } from "@/features/home/profile"
import { rulePath } from "@/features/path/components/rule-card"
import { cn } from "@/lib/utils"

import type {
  AssessmentRange,
  AssessmentResult,
  KnowledgeOutcome,
} from "./schemas"
import { AssessmentHeading } from "./start"

export function displayRange(range: AssessmentRange): string {
  return range.lower === range.upper
    ? range.lower
    : `${range.lower}–${range.upper}`
}

const outcomeTones: Record<KnowledgeOutcome["status"], Tone> = {
  demonstrated: "mastery",
  needs_practice: "learning",
  needs_more_evidence: "review",
  not_assessed: "muted",
}

export function AssessmentResults({
  result,
  language,
  languageName,
  partial,
  onContinue,
  onChoose,
}: {
  result: AssessmentResult
  language: string
  languageName: string
  partial: boolean
  onContinue: () => void
  onChoose?: () => void
}) {
  const t = useTranslations("assessment")
  const estimates = new Map(
    result.skillEstimates.map((estimate) => [estimate.skill, estimate]),
  )
  const range = result.difficultyRange
  const assessed = result.knowledgeOutcomes.filter(
    (item) => item.evidenceIds.length > 0,
  )
  const hasAdequateEvidence =
    result.evidenceSufficiency === "sufficient_for_screening"

  return (
    <div className="space-y-5">
      <AssessmentHeading
        title={t("resultsTitle")}
        description={t(partial ? "partialSubtitle" : "resultSubtitle", {
          language: languageName,
        })}
        results
      />
      <p className="text-muted-foreground text-sm">{t("grammarEvidence")}</p>
      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="space-y-4">
          <section className={cn(card, "space-y-2 p-5")}>
            <div className="flex flex-wrap items-center gap-2">
              <p className="text-muted-foreground text-xs font-medium">
                {t("recommendation")}
              </p>
              <Pill tone={hasAdequateEvidence ? "neutral" : "review"}>
                {t(`evidence.${result.evidenceSufficiency}`)}
              </Pill>
            </div>
            <h2 className="text-2xl font-semibold tracking-tight">
              {range ? displayRange(range) : t("noRange")}
            </h2>
            <p className="text-muted-foreground text-sm">
              {range ? t("estimateUse") : t("noRangeNote")}
            </p>
            {result.conflictingEvidence && (
              <p className="text-muted-foreground text-sm">
                {t("conflicting")}
              </p>
            )}
            {range?.boundary === "ceiling" && (
              <p className="text-muted-foreground text-xs">{t("ceiling")}</p>
            )}
            {range?.boundary === "floor" && (
              <p className="text-muted-foreground text-xs">{t("floor")}</p>
            )}
          </section>
          <section className={cn(card, "space-y-2 p-5")}>
            <h2 className="font-semibold">{t("skills")}</h2>
            <p className="text-muted-foreground text-xs">{t("skillsNote")}</p>
            <dl className="divide-border divide-y text-sm">
              {skillIds.map((skill) => {
                const estimate = estimates.get(skill)

                return (
                  <div
                    key={skill}
                    className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2"
                  >
                    <dt>{t(`skill.${skill}`)}</dt>
                    <dd className="flex flex-wrap items-baseline justify-end gap-x-3 gap-y-1">
                      {estimate?.difficultyRange && (
                        <b>{displayRange(estimate.difficultyRange)}</b>
                      )}
                      <span className="text-muted-foreground text-xs">
                        {!estimate
                          ? t(
                              result.unassessedSkills.includes(skill) &&
                                skill === "grammar"
                                ? "evidence.insufficient"
                                : "notAssessed",
                            )
                          : t("evidenceCount", {
                              count: estimate.usableEvidence,
                            })}
                      </span>
                    </dd>
                  </div>
                )
              })}
            </dl>
          </section>
          <div className="grid gap-4 sm:grid-cols-2">
            <OutcomeSummary
              title={t("strengths")}
              empty={t("noStrengths")}
              items={result.demonstratedStrengths}
              showCheck
            />
            <OutcomeSummary
              title={t("gaps")}
              empty={t("noGaps")}
              items={result.knowledgeGaps}
            />
          </div>
          {assessed.length > 0 && (
            <section className={card}>
              <div className="p-5 pb-3">
                <h2 className="font-semibold">{t("knowledge")}</h2>
                <p className="text-muted-foreground mt-1 text-xs">
                  {t("knowledgeNote")}
                </p>
              </div>
              {assessed.map((item) => (
                <details
                  key={item.knowledgeItemKey}
                  className="group border-border border-t"
                >
                  <summary className="flex min-h-12 cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1 px-5 py-2 text-sm [&::-webkit-details-marker]:hidden">
                    <span className="min-w-35 flex-1 font-medium">
                      {item.learningObjective}
                    </span>
                    <Pill tone={outcomeTones[item.status]}>
                      {t(`outcome.${item.status}`)}
                    </Pill>
                    <IconChevronDown
                      aria-hidden
                      size={16}
                      className="text-muted-foreground shrink-0 group-open:rotate-180"
                    />
                  </summary>
                  <div className="space-y-2 px-5 pb-4 text-sm">
                    <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                      <dt className="text-muted-foreground">
                        {t("categoryLabel")}
                      </dt>
                      <dd>
                        {t.has(`category.${item.category}`)
                          ? t(`category.${item.category}`)
                          : item.category.replaceAll("_", " ")}
                      </dd>
                      <dt className="text-muted-foreground">
                        {t("evidenceLabel")}
                      </dt>
                      <dd>
                        {t("answerCount", { count: item.evidenceIds.length })}
                      </dd>
                      <dt className="text-muted-foreground">
                        {t("actionLabel")}
                      </dt>
                      <dd>{t(`action.${item.status}`)}</dd>
                    </dl>
                    <Link
                      href={rulePath(
                        encodeURIComponent(item.knowledgeItemKey),
                        language,
                      )}
                      className="text-foreground underline underline-offset-4"
                    >
                      {t("openRule")}
                    </Link>
                  </div>
                </details>
              ))}
            </section>
          )}
        </div>
        <aside className="space-y-4">
          <section className={cn(card, "space-y-3 p-5")}>
            <h2 className="font-semibold">{t("focus")}</h2>
            {result.recommendedLearningObjectives.length ? (
              <ObjectiveList items={result.recommendedLearningObjectives} />
            ) : (
              <p className="text-muted-foreground text-sm">
                {t(range ? "focusRange" : "fundamentals")}
              </p>
            )}
            <Button
              className="h-11 w-full rounded-lg whitespace-normal"
              onClick={onContinue}
            >
              {t("continueToPath")}
            </Button>
          </section>
          {result.recommendedKnowledgeChecks.length > 0 && (
            <section className={cn(card, "space-y-3 p-5")}>
              <h2 className="font-semibold">{t("additionalChecks")}</h2>
              <ObjectiveList items={result.recommendedKnowledgeChecks} />
            </section>
          )}
          {onChoose && (
            <Button
              variant="outline"
              className="h-11 rounded-lg whitespace-normal"
              onClick={onChoose}
            >
              {t("otherStart")}
            </Button>
          )}
        </aside>
      </div>
    </div>
  )
}

function ObjectiveList({
  items,
  showCheck,
}: {
  items: KnowledgeOutcome[]
  showCheck?: boolean
}) {
  const t = useTranslations("assessment")
  return (
    <ul className={cn("space-y-1.5 text-sm", !showCheck && "list-disc pl-4")}>
      {items.map((item) => (
        <li
          key={item.knowledgeItemKey}
          className={cn(showCheck && "flex items-start gap-2")}
        >
          {showCheck && (
            <IconCheck
              size={16}
              aria-hidden
              className="text-success mt-0.5 shrink-0"
            />
          )}
          {item.learningObjective}
          {item.gapSupport === "suspected" && (
            <span className="text-muted-foreground text-xs"> · {t("suspectedGap")}</span>
          )}
        </li>
      ))}
    </ul>
  )
}

function OutcomeSummary({
  title,
  empty,
  items,
  showCheck,
}: {
  title: string
  empty: string
  items: KnowledgeOutcome[]
  showCheck?: boolean
}) {
  return (
    <section className={cn(card, "space-y-2 p-5")}>
      <h2 className="font-semibold">{title}</h2>
      {items.length ? (
        <ObjectiveList items={items} showCheck={showCheck} />
      ) : (
        <p className="text-muted-foreground text-sm">{empty}</p>
      )}
    </section>
  )
}
