"use client"

import { useEffect, useState } from "react"
import { usePathname } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { IconCheck, IconChevronLeft } from "@tabler/icons-react"
import { usePath, useTeachers } from "../hooks"
import { useReturnPoint } from "../return-point"
import { ExplainSwitch } from "./explain-switch"
import { pickableLevels } from "./level-scale"
import { ProgramView } from "./program-view"
import { StartError } from "./start-error"
import { TeachersList, type TeacherPair } from "./teachers-list"
import { TestRunner } from "./test-runner"

/**
 * The Teacher page: the learner's teachers side by side, and the program of
 * the one they open. Embedded above the teacher chat the pair comes from the
 * picked topic and there is no list: it is that one teacher's program, and
 * onTestChange tells the host to get out of the way while a test runs.
 */
export function PathScreen({
  language,
  translationLanguage,
  embedded = false,
  onTestChange,
}: {
  language?: string
  translationLanguage?: string
  embedded?: boolean
  onTestChange?: (testActive: boolean) => void
}) {
  if (embedded || language) {
    return (
      <Program
        language={language ?? "en"}
        nativeLanguage={translationLanguage ?? "ru"}
        embedded={embedded}
        onTestChange={onTestChange}
      />
    )
  }

  return <Teachers onTestChange={onTestChange} />
}

function Teachers({ onTestChange }: { onTestChange?: (testActive: boolean) => void }) {
  // undefined: nothing was picked here yet, so a teacher whose rule book the
  // learner came back from is the one shown; null: the list was chosen.
  const [picked, setPicked] = useState<TeacherPair | null | undefined>(undefined)
  const pathname = usePathname()
  const left = useReturnPoint()
  const selected =
    picked !== undefined
      ? picked
      : left && left.href.split("?")[0] === pathname
        ? { language: left.language, translationLanguage: left.translationLanguage }
        : null

  if (!selected) return <TeachersList onOpen={setPicked} />

  return (
    <Program
      language={selected.language}
      nativeLanguage={selected.translationLanguage}
      onBack={() => setPicked(null)}
      onTestChange={onTestChange}
    />
  )
}

function Program({
  language,
  nativeLanguage,
  embedded = false,
  onBack,
  onTestChange,
}: {
  language: string
  nativeLanguage: string
  embedded?: boolean
  onBack?: () => void
  onTestChange?: (testActive: boolean) => void
}) {
  const t = useTranslations("path")
  const locale = useLocale()
  const [testId, setTestId] = useState<string | null>(null)
  // The test on screen is a level picked from the reached ones, not a taken test.
  const [chosen, setChosen] = useState(false)
  const path = usePath(language, nativeLanguage)
  const { teachers } = useTeachers()
  // What the teacher explains in now, which is not always the learner's own language.
  const explainLanguage =
    teachers.data?.find((teacher) => teacher.language === language)?.explainLanguage ??
    nativeLanguage
  const languageNames = new Intl.DisplayNames([locale], { type: "language" })
  const languageName = languageNames.of(language) ?? language
  const active = path.active.data
  const pickable = pickableLevels(path.history.data ?? [], language)

  useEffect(() => {
    onTestChange?.(testId !== null)
  }, [testId, onTestChange])

  if (testId) {
    return (
      <TestRunner
        id={testId}
        chosen={chosen}
        onNextLevel={(level) => {
          // The finished test is dropped first so the new one mounts fresh.
          setTestId(null)
          setChosen(false)
          path.startPlacement.mutate(level, {
            onSuccess: (test) => setTestId(test.id),
          })
        }}
        onClose={(analyzed) => {
          setTestId(null)
          setChosen(false)
          if (analyzed) path.active.refetch()
        }}
      />
    )
  }

  return (
    <div
      className={
        embedded
          ? "anim-fade space-y-4"
          : "anim-fade space-y-4 lg:mx-auto lg:w-full lg:max-w-[704px]"
      }
    >
      {!embedded && (
        <header className="space-y-1">
          {onBack && (
            <Button variant="ghost" size="sm" onClick={onBack} className="-ml-2 h-auto px-2 py-1 text-xs">
              <IconChevronLeft size={14} />
              {t("backToTeachers")}
            </Button>
          )}
          <h1 className="text-[22px] font-semibold tracking-[-0.01em] lg:text-2xl">
            {t("teacherTitle", { language: languageName })}
          </h1>
          <p className="text-muted-foreground text-xs leading-relaxed lg:text-sm">
            {t("explainedIn", { language: languageNames.of(explainLanguage) ?? explainLanguage })}
          </p>
        </header>
      )}

      {!embedded && <ExplainSwitch language={language} />}

      {path.active.isPending ? (
        <div className="space-y-3">
          <Skeleton className="h-28 w-full rounded-lg" />
          <Skeleton className="h-40 w-full rounded-lg" />
        </div>
      ) : active ? (
        <ProgramView
          path={active}
          onStartFinal={() =>
            path.startFinal.mutate(active.translationLanguage, {
              onSuccess: (test) => setTestId(test.id),
            })
          }
          onReset={() => path.reset.mutate()}
          startError={path.startFinal.isError ? path.startFinal.error.message : null}
          starting={path.startFinal.isPending}
          asOf={path.active.dataUpdatedAt}
        />
      ) : (
        <>
          {pickable.length > 0 && (
            <section className="bg-background border-border space-y-3 rounded-lg border p-5">
              <div className="space-y-1.5">
                <h2 className="text-base font-semibold">{t("pickTitle")}</h2>
                <p className="text-muted-foreground max-w-[65ch] text-[13px] leading-relaxed">
                  {t("pickHint")}
                </p>
              </div>
              <div role="group" aria-label={t("pickTitle")} className="flex flex-wrap gap-2">
                {pickable.map((level) => (
                  <Button
                    key={level}
                    variant="outline"
                    size="lg"
                    disabled={path.chooseLevel.isPending}
                    aria-label={t("pickLevel", { level })}
                    onClick={() =>
                      path.chooseLevel.mutate(level, {
                        onSuccess: (test) => {
                          setChosen(true)
                          setTestId(test.id)
                        },
                      })
                    }
                    className="min-w-14 px-4 text-[13px] font-semibold"
                  >
                    {level}
                  </Button>
                ))}
              </div>
              {path.chooseLevel.isError && (
                <p role="alert" className="text-destructive text-xs">
                  {path.chooseLevel.error.message}
                </p>
              )}
            </section>
          )}
          <section className="bg-background border-border space-y-3 rounded-lg border p-5">
            <div className="space-y-2.5">
              <h2 className="text-base font-semibold">{t("placementTitle")}</h2>
              <ul className="max-w-[65ch] space-y-2 text-[15px] leading-[1.6]">
                {(["Writer", "Ladder", "Mistakes"] as const).map((point) => (
                  <li key={point} className="flex gap-2.5">
                    <IconCheck
                      size={16}
                      aria-hidden="true"
                      className="text-success mt-1 shrink-0"
                    />
                    <span>{t(`placementPoint${point}`)}</span>
                  </li>
                ))}
              </ul>
              <p className="bg-muted text-muted-foreground inline-block rounded-full px-3 py-1 text-xs font-medium">
                {t("placementNeeds")}
              </p>
            </div>
            {path.startPlacement.isError && (
              <StartError message={path.startPlacement.error.message} />
            )}
            <Button
              onClick={() =>
                path.startPlacement.mutate(undefined, {
                  onSuccess: (test) => setTestId(test.id),
                })
              }
              disabled={path.startPlacement.isPending}
              size="lg"
              className="px-4 text-[13px]"
            >
              {t(path.startPlacement.isPending ? "starting" : "startPlacement")}
            </Button>
          </section>
        </>
      )}

      <History language={language} nativeLanguage={nativeLanguage} />
    </div>
  )
}

function History({ language, nativeLanguage }: { language: string; nativeLanguage: string }) {
  const t = useTranslations("path")
  const { history, clearHistory } = usePath(language, nativeLanguage)
  const [confirming, setConfirming] = useState(false)

  const shown = (history.data ?? []).filter((item) => item.language === language)

  if (shown.length === 0) return null

  return (
    <section className="space-y-2.5">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">{t("history")}</h2>
        {!confirming && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setConfirming(true)}
            className="text-muted-foreground -mr-2 h-auto px-2 py-1 text-xs"
          >
            {t("historyClear")}
          </Button>
        )}
      </div>
      {confirming && (
        <div
          role="alertdialog"
          aria-label={t("historyClear")}
          className="border-destructive/30 bg-destructive/5 space-y-2.5 rounded-lg border p-4"
        >
          <p className="text-[13px] leading-relaxed">{t("historyClearConfirm")}</p>
          {clearHistory.isError && (
            <p role="alert" className="text-destructive text-xs">
              {clearHistory.error.message}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button
              variant="destructive"
              size="lg"
              disabled={clearHistory.isPending}
              onClick={() =>
                clearHistory.mutate(undefined, { onSuccess: () => setConfirming(false) })
              }
              className="px-4 text-[13px]"
            >
              {t(clearHistory.isPending ? "historyClearing" : "historyClearYes")}
            </Button>
            <Button
              variant="outline"
              size="lg"
              disabled={clearHistory.isPending}
              onClick={() => setConfirming(false)}
              className="px-4 text-[13px]"
            >
              {t("cancel")}
            </Button>
          </div>
        </div>
      )}
      <ul className="space-y-2">
        {shown.map((item) => (
          <li
            key={item.id}
            className="bg-background border-border flex items-center gap-3 rounded-lg border p-4"
          >
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold">
                {item.language.toUpperCase()} · {t("level")} {item.level}
              </p>
              <p className="text-muted-foreground text-xs">
                {t(`source.${item.source}`)} · {t(`status.${item.status}`)} ·{" "}
                {new Date(item.createdAt).toLocaleDateString()}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
