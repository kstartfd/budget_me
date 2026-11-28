"use client"

import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { claimableLevels, type Profile } from "@/features/home/profile"
import { Pill, card } from "@/features/home/components/parts"
import { cn } from "@/lib/utils"

export const startOptions = [
  "quick",
  "full",
  "skill",
  "manual",
  "scratch",
] as const
export type StartOption = (typeof startOptions)[number]

export function AssessmentHeading({
  title,
  description,
  results,
}: {
  title: string
  description?: string
  results?: boolean
}) {
  const t = useTranslations("assessment")

  return (
    <header className="space-y-1">
      <p className="text-muted-foreground text-xs">
        {t(results ? "resultsCrumb" : "crumb")}
      </p>
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {description && (
        <p className="text-muted-foreground max-w-[760px] text-sm">
          {description}
        </p>
      )}
    </header>
  )
}

export function AssessmentStart({
  option,
  onOption,
  level,
  onLevel,
  quickAvailable,
  onContinue,
}: {
  option: StartOption
  onOption: (option: StartOption) => void
  level: Profile["level"]
  onLevel: (next: Profile["level"]) => void
  quickAvailable: boolean
  onContinue: () => void
}) {
  const t = useTranslations("assessment")

  return (
    <div className="space-y-5">
      <AssessmentHeading
        title={t("selectTitle")}
        description={t("selectText")}
      />
      <fieldset className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,300px),1fr))] gap-3">
        <legend className="sr-only">{t("type")}</legend>
        {startOptions.map((id) => {
          const disabled =
            id === "full" ||
            id === "skill" ||
            (id === "quick" && !quickAvailable)
          const selected = option === id

          return (
            <label
              key={id}
              className={cn(
                card,
                "relative flex min-h-28 cursor-pointer flex-col gap-1.5 p-4 transition-colors has-focus-visible:ring-2",
                selected && "border-foreground bg-muted ring-1 ring-foreground",
                disabled && "cursor-not-allowed bg-muted/60 opacity-65",
              )}
            >
              <input
                type="radio"
                name="assessment-type"
                className="sr-only"
                value={id}
                checked={selected}
                disabled={disabled}
                onChange={() => onOption(id)}
              />
              <span className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                <span className="text-base font-semibold">
                  {t(`option.${id}.title`)}
                </span>
                <Pill>
                  {disabled ? t("unavailable") : t(`option.${id}.badge`)}
                </Pill>
              </span>
              <span className="text-muted-foreground text-sm">
                {t(`option.${id}.text`)}
              </span>
              {!disabled && (
                <span className="text-muted-foreground text-xs">
                  {t(`option.${id}.meta`)}
                </span>
              )}
            </label>
          )
        })}
      </fieldset>
      {option === "manual" && (
        <div className={cn(card, "max-w-[560px] space-y-2 p-4")}>
          <label className="flex flex-wrap items-center justify-between gap-2 text-sm font-medium">
            {t("approximate")}
            <select
              value={level}
              onChange={(event) =>
                onLevel(event.target.value as Profile["level"])
              }
              className="border-border bg-background h-10 min-w-24 rounded-md border px-3 text-sm"
            >
              {claimableLevels.map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          </label>
          <p className="text-muted-foreground text-xs">{t("selfReport")}</p>
        </div>
      )}
      {!quickAvailable && (
        <p className="text-muted-foreground text-sm">{t("quickUnavailable")}</p>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button className="h-11 rounded-lg px-4 text-sm" onClick={onContinue}>
          {t(
            option === "manual"
              ? "saveStart"
              : option === "scratch"
                ? "startScratch"
                : "continue",
          )}
        </Button>
        <p className="text-muted-foreground text-xs">{t("optional")}</p>
      </div>
    </div>
  )
}

export function AssessmentIntro({
  language,
  onBack,
  onStart,
  busy,
  error,
}: {
  language: string
  onBack: () => void
  onStart: () => void
  busy: boolean
  error?: string
}) {
  const t = useTranslations("assessment")

  return (
    <section className={cn(card, "max-w-[720px]")}>
      <header className="border-border space-y-1 border-b px-5 py-5 sm:px-6">
        <p className="text-muted-foreground text-xs">
          {t("teacher", { language })}
        </p>
        <h1 className="text-[22px] font-semibold">{t("option.quick.title")}</h1>
      </header>
      <div className="space-y-4 px-5 py-5 sm:px-6">
        <dl className="bg-muted grid gap-3 rounded-lg p-3.5 text-sm sm:grid-cols-3">
          {(["coverage", "workload", "questionCount"] as const).map((field) => (
            <div key={field} className="space-y-1">
              <dt className="text-muted-foreground text-xs">{t(field)}</dt>
              <dd className="font-medium">{t(`${field}Value`)}</dd>
            </div>
          ))}
        </dl>
        <ul className="text-muted-foreground list-disc space-y-1.5 pl-5 text-sm">
          {(["skip", "adaptive", "feedback", "save", "estimate"] as const).map(
            (id) => (
              <li key={id}>{t(`instruction.${id}`)}</li>
            ),
          )}
        </ul>
        {error && (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        )}
      </div>
      <footer className="border-border flex flex-wrap justify-between gap-2 border-t px-5 py-4 sm:px-6">
        <Button
          variant="outline"
          className="h-11 rounded-lg px-4 text-sm"
          onClick={onBack}
          disabled={busy}
        >
          {t("back")}
        </Button>
        <Button
          className="h-11 rounded-lg px-4 text-sm"
          onClick={onStart}
          disabled={busy}
        >
          {t(busy ? "preparing" : "start")}
        </Button>
      </footer>
    </section>
  )
}
