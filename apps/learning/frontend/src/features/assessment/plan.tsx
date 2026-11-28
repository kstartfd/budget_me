"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { useAiSettings } from "@/features/ai"
import { AiNotice } from "@/features/home/components/dashboard"
import { useLanguageName } from "@/features/home/components/language-switcher"
import { card } from "@/features/home/components/parts"
import {
  GeneratingPanel,
  useBuild,
} from "@/features/home/components/setup-panel"
import { defaultProfile, goalIds, useProfile } from "@/features/home/profile"
import type { PathTeacher, PathTest } from "@/features/path/schemas"
import { cn } from "@/lib/utils"

import { displayRange } from "./results"

export function AssessmentPlan({
  teacher,
  test,
  onBack,
}: {
  teacher: PathTeacher
  test?: PathTest
  onBack: () => void
}) {
  const t = useTranslations("assessment")
  const home = useTranslations("home")
  const router = useRouter()
  const ai = useAiSettings()
  const build = useBuild(teacher)
  const [stored, saveProfile] = useProfile(teacher.language)
  const profile = stored ?? defaultProfile
  const name = useLanguageName()
  const result = test?.assessment?.result
  const needsAi = !test
  const range = result?.difficultyRange
  const manual = !test && profile.start === "manual"
  const startingPoint = range
    ? displayRange(range)
    : manual
      ? `${profile.level} · ${t("option.manual.badge")}`
      : t("fundamentals")
  const focus = result?.recommendedLearningObjectives
    .map((item) => item.learningObjective)
    .join(", ")

  if (build.isPending || build.isSuccess)
    return <GeneratingPanel teacher={teacher} />

  return (
    <div className="max-w-[760px] space-y-4">
      {needsAi && !ai.isPending && !ai.data && <AiNotice />}
      <section className={cn(card, "divide-border divide-y")}>
        <header className="space-y-1 p-5 sm:p-6">
          <h1 className="text-[22px] font-semibold">
            {t("planTitle", { language: name(teacher.language) })}
          </h1>
          <p className="text-muted-foreground text-sm">{t("planText")}</p>
        </header>
        <dl className="grid gap-x-6 gap-y-2 p-5 text-sm sm:grid-cols-[auto_1fr] sm:p-6">
          <dt className="text-muted-foreground">{t("recommendation")}</dt>
          <dd className="font-medium">{startingPoint}</dd>
          {focus && (
            <>
              <dt className="text-muted-foreground">{t("focus")}</dt>
              <dd>{focus}</dd>
            </>
          )}
          <dt className="text-muted-foreground">{home("sumGoals")}</dt>
          <dd>
            {profile.goals.map((id) => home(`goal.${id}`)).join(", ") ||
              home("noneShort")}
          </dd>
          <dt className="text-muted-foreground">{home("sumRoutine")}</dt>
          <dd>
            {home("routineValue", {
              hours: profile.weekly,
              minutes: profile.session,
            })}
          </dd>
          <dt className="text-muted-foreground">{home("sumExplanations")}</dt>
          <dd>{name(teacher.explainLanguage)}</dd>
        </dl>
        <div className="space-y-4 p-5 sm:p-6">
          {!teacher.path && (
            <details className="bg-muted rounded-lg px-3.5 text-sm">
              <summary className="cursor-pointer py-2.5 font-medium">
                {t("adjustGoals")}
              </summary>
              <fieldset className="grid gap-2 pb-3 sm:grid-cols-2">
                <legend className="sr-only">{home("sumGoals")}</legend>
                {goalIds.map((goal) => (
                  <label
                    key={goal}
                    className="flex min-h-8 items-center gap-2.5"
                  >
                    <input
                      className="accent-foreground size-4"
                      type="checkbox"
                      checked={profile.goals.includes(goal)}
                      onChange={(event) =>
                        saveProfile({
                          ...profile,
                          goals: event.target.checked
                            ? [...profile.goals, goal]
                            : profile.goals.filter((id) => id !== goal),
                        })
                      }
                    />
                    {home(`goal.${goal}`)}
                  </label>
                ))}
              </fieldset>
            </details>
          )}
          {build.isError && (
            <p role="alert" className="text-destructive text-sm">
              {build.error.message}
            </p>
          )}
          <div className="flex flex-wrap justify-between gap-2">
            <Button
              variant="outline"
              className="h-11 rounded-lg px-4 text-sm"
              onClick={onBack}
            >
              {t(test ? "review" : "back")}
            </Button>
            {teacher.path ? (
              <Button asChild className="h-11 rounded-lg px-4 text-sm">
                <Link
                  href={`/learning/path?language=${encodeURIComponent(teacher.language)}`}
                >
                  {t("openPath")}
                </Link>
              </Button>
            ) : (
              <Button
                className="h-11 rounded-lg px-4 text-sm"
                disabled={needsAi && !ai.data}
                onClick={() =>
                  build.mutate(
                    test
                      ? { kind: "test", testId: test.id }
                      : manual
                        ? { kind: "claim", level: profile.level }
                        : { kind: "scratch" },
                    { onSuccess: () => router.push("/learning/overview") },
                  )
                }
              >
                {t(build.isError ? "retry" : "build")}
              </Button>
            )}
          </div>
        </div>
      </section>
    </div>
  )
}
