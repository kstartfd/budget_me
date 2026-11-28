"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { IconCheck } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { useAiSettings } from "@/features/ai"
import { hasQuickAssessment } from "@/features/assessment/availability"
import { useAuth } from "@/features/auth"
import * as api from "@/features/path/api"
import { useTeachers } from "@/features/path/hooks"
import { languages } from "@/features/path/languages"
import type { PathTeacher } from "@/features/path/schemas"
import { cn } from "@/lib/utils"

import {
  claimableLevels,
  defaultProfile,
  goalIds,
  preferencesOf,
  prefIds,
  sessionOptions,
  setupSteps,
  skillIds,
  startIds,
  useActiveLanguage,
  useProfile,
  useSetupMemory,
  weeklyOptions,
  type GoalId,
  type PrefId,
  type Profile,
  type SkillId,
  type StartId,
} from "../profile"
import { useLanguageName } from "./language-switcher"
import { Choice, Chip, card, Segmented } from "./parts"

/** Suggest the interface language for a learner who has no profile yet. */
function guessNative(locale: string): string {
  const code = locale.split("-")[0]

  return ["en", "es", "ru"].includes(code) ? code : "ru"
}

/**
 * The five steps set up the learner's native language, study language,
 * goals, routine and starting point.
 * The profile is saved before anything that needs the learner's AI provider, so
 * a failure there loses nothing.
 */
export function Wizard() {
  const { teachers } = useTeachers()
  const locale = useLocale()
  const t = useTranslations("home")
  const first = teachers.data?.[0] ?? null
  const [existing] = useProfile(first?.language ?? null)

  if (!teachers.data && teachers.isError) {
    return (
      <div className="space-y-3">
        <p role="alert" className="text-destructive text-sm">{teachers.error.message}</p>
        <Button variant="outline" onClick={() => void teachers.refetch()}>
          {t("retry")}
        </Button>
      </div>
    )
  }

  if (!teachers.data) {
    return (
      <div className="space-y-4 lg:max-w-[880px]">
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-72 w-full rounded-xl" />
      </div>
    )
  }

  return (
    <WizardForm
      taken={teachers.data.map((teacher) => teacher.language)}
      initialNative={
        first && languages.includes(first.translationLanguage)
          ? first.translationLanguage
          : guessNative(locale)
      }
      initial={existing ?? defaultProfile}
    />
  )
}

function WizardForm({
  taken,
  initialNative,
  initial,
}: {
  taken: string[]
  initialNative: string
  initial: Profile
}) {
  const t = useTranslations("home")
  const name = useLanguageName()
  const router = useRouter()
  const client = useQueryClient()
  const { user } = useAuth()
  const ai = useAiSettings()
  const [, setActiveLanguage] = useActiveLanguage()
  const [fresh] = useState(taken.length === 0)

  const [step, setStep] = useState(1)
  const [native, setNative] = useState(initialNative)
  const choices = languages.filter((code) => code !== native && !taken.includes(code))
  const [chosen, setChosen] = useState<string | null>(null)
  const [savedTeacher, setSavedTeacher] = useState<PathTeacher | null>(null)
  const [preparedTestId, setPreparedTestId] = useState<string | null>(null)
  const language =
    savedTeacher?.language ??
    (chosen && choices.includes(chosen) ? chosen : choices[0] ?? "")
  const aiReady = ai.data != null
  const [explainInTarget, setExplainInTarget] = useState(false)
  const [profile, setProfile] = useState<Profile>({ ...initial, start: initial.start ?? "assess" })

  const [, saveProfile] = useProfile(language || null)
  const [, saveMemory] = useSetupMemory(language || null)
  const start = profile.start ?? "assess"

  const patch = (next: Partial<Profile>) => setProfile((prev) => ({ ...prev, ...next }))
  const toggle = <K extends "goals" | "skills" | "prefs">(key: K, id: Profile[K][number]) =>
    setProfile((prev) => {
      const list = prev[key] as string[]
      const next = list.includes(id) ? list.filter((item) => item !== id) : [...list, id]

      return { ...prev, [key]: next }
    })

  // Keep the teacher and the prepared test through retries after a partial save.
  const finish = useMutation({
    mutationFn: async () => {
      const teacher = savedTeacher ?? (await api.createTeacher(language, native))
      setSavedTeacher(teacher)

      saveProfile({ ...profile, start })
      setActiveLanguage(language)

      if (teacher.explainInTarget !== explainInTarget) {
        const updated = await api.updateTeacher(teacher.id, explainInTarget)
        setSavedTeacher(updated)
      }

      if (start === "assess" && hasQuickAssessment(language)) return "assessment" as const
      if (!aiReady) return "saved" as const

      if (start === "assess") {
        const test = await api.startPlacement(language, teacher.translationLanguage)
        saveMemory({ testId: test.id })
        return "resume" as const
      }

      let testId = preparedTestId
      if (!testId) {
        const test = await api.chooseLevel(
          language,
          teacher.translationLanguage,
          start === "manual" ? profile.level : "A1",
          true,
        )
        testId = test.id
        setPreparedTestId(testId)
        saveMemory({ testId })
      }

      const job = await api.startBuild(testId, preferencesOf(profile))
      saveMemory({ testId, jobId: job.id })

      return "build" as const
    },
    onSuccess: async (next) => {
      await client.invalidateQueries({ queryKey: ["path", user?.id] })
      router.push(next === "assessment"
        ? "/learning/assessment"
        : next === "resume" ? `/learning/overview?resume=${language}` : "/learning/overview")
    },
    onError: () => client.invalidateQueries({ queryKey: ["path", user?.id] }),
  })

  const last = step === setupSteps.length
  const startLocked = finish.isPending || savedTeacher !== null
  const finishLabel = finish.isPending
    ? "creating"
    : finish.isError
      ? "retry"
      : aiReady || (start === "assess" && hasQuickAssessment(language))
        ? `finish.${start}` as const
        : "saveProfile"
  const goalNames = profile.goals.map((id) => t(`goal.${id}`)).join(", ")
  const skillNames = profile.skills.map((id) => t(`skill.${id}`)).join(", ")
  const nativeName = name(native)
  const learned = language ? name(language) : ""
  const choosingNative = step === 1
  const languageOptions = choosingNative ? languages : choices
  const selectedLanguage = choosingNative ? native : language
  const selectLanguage = choosingNative ? setNative : setChosen

  const summary: [string, string][] = [
    [t("step.native"), nativeName],
    [t("sumLanguage"), learned],
    [t("sumExplanations"), explainInTarget ? learned : nativeName],
    [t("sumGoals"), goalNames || t("noneSelected")],
    [t("sumSkills"), skillNames || t("noneShort")],
    [t("sumRoutine"), t("routineValue", { hours: profile.weekly, minutes: profile.session })],
    [
      t("sumStart"),
      start === "manual"
        ? t("startValue.manual", { level: profile.level })
        : t(`startValue.${start}`),
    ],
  ]

  return (
    <div className="anim-fade space-y-6">
      <header>
        <p className="text-muted-foreground mb-1 text-xs">{t("crumb", { page: t("wizardCrumb") })}</p>
        <h1 className="text-2xl font-semibold tracking-[-0.01em]">
          {t(fresh ? "wizardFirst" : "wizardMore")}
        </h1>
        <p className="text-muted-foreground mt-0.5 text-sm">{t("wizardSub")}</p>
      </header>

      <section className={cn(card, "max-w-[880px]")}>
        <ol
          aria-label={t("stepperLabel")}
          className="border-border flex flex-wrap gap-x-5 gap-y-2 border-b px-6 py-4 text-[13px]"
        >
          {setupSteps.map((id, index) => {
            const number = index + 1
            const done = number < step
            const current = number === step

            return (
              <li
                key={id}
                aria-current={current ? "step" : undefined}
                className={cn(
                  "flex items-center gap-1.5",
                  current ? "text-foreground font-semibold" : done ? "text-foreground" : "text-muted-foreground",
                )}
              >
                <span
                  className={cn(
                    "flex size-5 items-center justify-center rounded-full border text-[11px]",
                    done
                      ? "border-foreground bg-foreground text-background"
                      : current
                        ? "border-foreground"
                        : "border-border",
                  )}
                >
                  {done ? <IconCheck size={12} aria-hidden="true" /> : number}
                </span>
                {t(`step.${id}`)}
              </li>
            )
          })}
        </ol>

        <div className="flex min-h-[260px] flex-col gap-4 p-6">
          {step <= 2 && (
            <>
              <StepHead
                title={t(choosingNative ? "nativeTitle" : "learningLanguageTitle")}
                hint={choosingNative ? t("nativeHint") : undefined}
              />
              {languageOptions.length > 0 ? (
                <div
                  role="radiogroup"
                  aria-label={t(choosingNative ? "step.native" : "step.language")}
                  className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,130px),1fr))] gap-2 sm:grid-cols-[repeat(auto-fit,minmax(150px,1fr))]"
                >
                  {languageOptions.map((code) => (
                    <Choice
                      key={code}
                      role="radio"
                      aria-checked={selectedLanguage === code}
                      selected={selectedLanguage === code}
                      onClick={() => selectLanguage(code)}
                      className="flex min-h-12 items-center justify-between font-medium"
                    >
                      {name(code)}
                      {selectedLanguage === code && <IconCheck size={16} aria-hidden="true" />}
                    </Choice>
                  ))}
                </div>
              ) : (
                <p className="text-muted-foreground text-sm">{t("noAvailableLanguages")}</p>
              )}
              {!choosingNative && language && (
                <div className="border-border flex items-start gap-3 border-t pt-4">
                  <Checkbox
                    id="wizard-explain-in-target"
                    className="mt-0.5"
                    aria-describedby="wizard-explanation-default"
                    checked={explainInTarget}
                    onCheckedChange={(checked) => setExplainInTarget(checked === true)}
                  />
                  <div>
                    <label htmlFor="wizard-explain-in-target" className="text-sm font-medium">
                      {t("explainInTarget", { language: learned })}
                    </label>
                    <p id="wizard-explanation-default" className="text-muted-foreground text-[13px]">
                      {t("explanationDefault", { language: nativeName })}
                    </p>
                  </div>
                </div>
              )}
            </>
          )}

          {step === 3 && (
            <>
              <StepHead title={t("s3Title")} hint={t("s3Hint")} />
              <div role="group" aria-label={t("goalsGroup")} className="flex flex-wrap gap-2">
                {goalIds.map((id: GoalId) => (
                  <Chip key={id} pressed={profile.goals.includes(id)} onClick={() => toggle("goals", id)}>
                    {profile.goals.includes(id) && "✓ "}
                    {t(`goal.${id}`)}
                  </Chip>
                ))}
              </div>
              <div>
                <p className="text-[13px] font-medium">
                  {t("skillsTitle")} <span className="text-muted-foreground font-normal">{t("optional")}</span>
                </p>
                <div role="group" aria-label={t("skillsTitle")} className="mt-2 flex flex-wrap gap-2">
                  {skillIds.map((id: SkillId) => (
                    <Chip key={id} pressed={profile.skills.includes(id)} onClick={() => toggle("skills", id)}>
                      {profile.skills.includes(id) && "✓ "}
                      {t(`skill.${id}`)}
                    </Chip>
                  ))}
                </div>
              </div>
            </>
          )}

          {step === 4 && (
            <>
              <StepHead title={t("s4Title")} hint={t("s4Hint")} />
              <div className="flex flex-wrap gap-6">
                <div className="min-w-60 flex-1 sm:flex-none">
                  <p className="mb-1.5 text-[13px]">{t("weekly")}</p>
                  <Segmented
                    label={t("weekly")}
                    options={weeklyOptions}
                    value={weeklyOptions.find((n) => n === profile.weekly) ?? weeklyOptions[1]}
                    onChange={(weekly) => patch({ weekly })}
                    format={(n) => t("hoursShort", { n })}
                  />
                </div>
                <div className="min-w-60 flex-1 sm:flex-none">
                  <p className="mb-1.5 text-[13px]">{t("session")}</p>
                  <Segmented
                    label={t("session")}
                    options={sessionOptions}
                    value={sessionOptions.find((n) => n === profile.session) ?? sessionOptions[1]}
                    onChange={(session) => patch({ session })}
                    format={(n) => t("minutesShort", { n })}
                  />
                </div>
              </div>
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-[13px]">{t("prefsLegend")}</legend>
                {prefIds.map((id: PrefId) => (
                  <label key={id} className="flex min-h-8 items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={profile.prefs.includes(id)}
                      onChange={() => toggle("prefs", id)}
                      className="size-[18px]"
                    />
                    {t(`pref.${id}`)}
                  </label>
                ))}
              </fieldset>
            </>
          )}

          {step === 5 && (
            <>
              <StepHead title={t("s5Title")} hint={t("s5Hint")} />
              <div role="radiogroup" aria-label={t("s5Group")} className="flex flex-col gap-2">
                {startIds.map((id: StartId) => (
                  <Choice
                    key={id}
                    role="radio"
                    aria-checked={start === id}
                    disabled={startLocked}
                    selected={start === id}
                    onClick={() => patch({ start: id })}
                  >
                    <span className="font-medium">
                      {start === id ? "● " : "○ "}
                      {t(`startOption.${id}.title`)}
                    </span>
                    <span className="text-muted-foreground block text-[13px]">{t(`startOption.${id}.text`)}</span>
                  </Choice>
                ))}
              </div>
              {start === "manual" && (
                <label className="block max-w-xs space-y-1 text-[13px]">
                  <span>{t("approxLevel")}</span>
                  <span className="text-muted-foreground block">{t("selfReported")}</span>
                  <Select
                    disabled={startLocked}
                    value={profile.level}
                    onValueChange={(level) => patch({ level: level as Profile["level"] })}
                  >
                    <SelectTrigger className="bg-background h-10 w-full px-3 text-sm" aria-label={t("approxLevel")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {claimableLevels.map((level) => (
                        <SelectItem key={level} value={level}>
                          {level}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </label>
              )}
              <div className="bg-muted rounded-[10px] px-4 py-3 text-[13px]">
                <p className="mb-1.5 font-semibold">{t("summary")}</p>
                <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                  {summary.map(([label, value]) => (
                    <div key={label} className="contents">
                      <dt className="text-muted-foreground">{label}</dt>
                      <dd>{value}</dd>
                    </div>
                  ))}
                </dl>
                <p className="text-muted-foreground mt-2">{t(`summaryNote.${start}`)}</p>
              </div>
              {ai.data === null && (
                <p role="status" className="text-muted-foreground text-[13px]">
                  {t("needsAiNote")}{" "}
                  <Link href="/learning/settings" className="text-foreground underline">
                    {t("configureAi")}
                  </Link>
                </p>
              )}
              {finish.isError && (
                <div role="alert" className="border-destructive/30 bg-destructive/5 space-y-1 rounded-lg border p-3 text-[13px]">
                  <p className="text-destructive">{finish.error.message}</p>
                  {savedTeacher && (
                    <p className="text-muted-foreground">
                      {t("wizardRetryHint")}{" "}
                      <Link href="/learning/overview" className="text-foreground underline">
                        {t("goToOverview")}
                      </Link>
                    </p>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        <div className="border-border flex flex-wrap justify-between gap-2 border-t px-6 py-4">
          <Button
            variant="ghost"
            size="lg"
            disabled={finish.isPending}
            onClick={() => router.push("/learning/overview")}
            className="text-muted-foreground h-10 px-3.5 text-sm underline"
          >
            {t("cancel")}
          </Button>
          <div className="flex gap-2">
            {step > 1 && (
              <Button
                variant="outline"
                size="lg"
                disabled={startLocked}
                onClick={() => setStep(step - 1)}
                className="h-10 px-4 text-sm"
              >
                {t("back")}
              </Button>
            )}
            <Button
              size="lg"
              disabled={finish.isPending || (step > 1 && !language)}
              onClick={() => (last ? finish.mutate() : setStep(step + 1))}
              className="h-10 px-4 text-sm"
            >
              {t(last ? finishLabel : "continue")}
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}

function StepHead({ title, hint }: { title: string; hint?: string }) {
  return (
    <div>
      <h2 className="text-lg font-semibold">{title}</h2>
      {hint && <p className="text-muted-foreground text-sm">{hint}</p>}
    </div>
  )
}
