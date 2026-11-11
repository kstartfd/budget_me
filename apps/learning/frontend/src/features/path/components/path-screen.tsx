"use client"

import { useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { IconChevronLeft } from "@tabler/icons-react"
import { usePath } from "../hooks"
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
  const [selected, setSelected] = useState<TeacherPair | null>(null)

  if (!selected) return <TeachersList onOpen={setSelected} />

  return (
    <Program
      language={selected.language}
      nativeLanguage={selected.translationLanguage}
      onBack={() => setSelected(null)}
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
  const path = usePath(language, nativeLanguage)
  const languageNames = new Intl.DisplayNames([locale], { type: "language" })
  const languageName = languageNames.of(language) ?? language
  const active = path.active.data

  useEffect(() => {
    onTestChange?.(testId !== null)
  }, [testId, onTestChange])

  if (testId) {
    return (
      <TestRunner
        id={testId}
        onNextLevel={(level) => {
          // The finished test is dropped first so the new one mounts fresh.
          setTestId(null)
          path.startPlacement.mutate(level, {
            onSuccess: (test) => setTestId(test.id),
          })
        }}
        onClose={(analyzed) => {
          setTestId(null)
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
            {t("explainedIn", { language: languageNames.of(nativeLanguage) ?? nativeLanguage })}
          </p>
        </header>
      )}

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
        />
      ) : (
        <section className="bg-background border-border space-y-3 rounded-lg border p-5">
          <p className="text-sm leading-relaxed">{t("placementIntro")}</p>
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
      )}

      <History language={language} nativeLanguage={nativeLanguage} />
    </div>
  )
}

function History({ language, nativeLanguage }: { language: string; nativeLanguage: string }) {
  const t = useTranslations("path")
  const { history } = usePath(language, nativeLanguage)

  if (!history.data || history.data.length === 0) return null

  return (
    <section className="space-y-2.5">
      <h2 className="text-sm font-semibold">{t("history")}</h2>
      <ul className="space-y-2">
        {history.data
          .filter((item) => item.language === language)
          .map((item) => (
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
