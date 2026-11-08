"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { usePath } from "../hooks"
import { ProgramView } from "./program-view"
import { TestRunner } from "./test-runner"

const languages = [
  "en",
  "es",
  "fr",
  "de",
  "it",
  "pt",
  "pl",
  "ru",
  "be",
  "uk",
  "zh",
  "ja",
  "ko",
  "tr",
]

/** The teacher tab: a test, a program built from it, level by level. */
export function PathScreen() {
  const t = useTranslations("path")
  const locale = useLocale()
  const [language, setLanguage] = useState("en")
  const [nativeLanguage, setNativeLanguage] = useState("ru")
  const [testId, setTestId] = useState<string | null>(null)
  const path = usePath(language, nativeLanguage)
  const languageNames = new Intl.DisplayNames([locale], { type: "language" })
  const active = path.active.data

  if (testId) {
    return (
      <TestRunner
        id={testId}
        onClose={(analyzed) => {
          setTestId(null)
          if (analyzed) path.active.refetch()
        }}
      />
    )
  }

  return (
    <div className="anim-fade space-y-[18px] lg:mx-auto lg:max-w-[760px] lg:space-y-[22px] lg:pt-5">
      <header className="space-y-1.5">
        <h1 className="text-[26px] font-bold tracking-[-0.02em] lg:text-[28px]">
          {t("title")}
        </h1>
        <p className="text-muted-foreground text-[13px] leading-relaxed lg:text-sm">
          {t("subtitle")}
        </p>
      </header>

      <div className="grid gap-3 sm:grid-cols-2">
        {(
          [
            [language, setLanguage, "language"],
            [nativeLanguage, setNativeLanguage, "nativeLanguage"],
          ] as const
        ).map(([value, setValue, name]) => (
          <label key={name} className="space-y-2 text-sm font-medium">
            <span>{t(name)}</span>
            <Select
              value={value}
              onValueChange={setValue}
              disabled={!!active}
            >
              <SelectTrigger className="bg-secondary border-border focus-visible:border-primary h-12 w-full rounded-2xl px-4 text-sm focus-visible:ring-0">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {languages.map((code) => (
                  <SelectItem key={code} value={code}>
                    {languageNames.of(code) ?? code}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
        ))}
      </div>

      {path.active.isPending ? (
        <div className="space-y-3">
          <Skeleton className="h-28 w-full rounded-[22px]" />
          <Skeleton className="h-40 w-full rounded-[22px]" />
        </div>
      ) : active ? (
        <ProgramView
          path={active}
          onStartFinal={() =>
            path.startFinal.mutate(undefined, {
              onSuccess: (test) => setTestId(test.id),
            })
          }
          onReset={() => path.reset.mutate()}
        />
      ) : (
        <section className="bg-card border-border space-y-3 rounded-[22px] border p-5 shadow-[var(--shadow-card)]">
          <p className="text-sm leading-relaxed">{t("placementIntro")}</p>
          {path.startPlacement.isError && (
            <p role="alert" className="text-destructive text-sm">
              {path.startPlacement.error.message}
            </p>
          )}
          <Button
            onClick={() =>
              path.startPlacement.mutate(undefined, {
                onSuccess: (test) => setTestId(test.id),
              })
            }
            disabled={path.startPlacement.isPending}
            className="text-primary-foreground h-auto rounded-[18px] px-5 py-3 text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
          >
            {t(path.startPlacement.isPending ? "starting" : "startPlacement")}
          </Button>
        </section>
      )}

      <History />
    </div>
  )
}

function History() {
  const t = useTranslations("path")
  const [language] = useState("en")
  const [nativeLanguage] = useState("ru")
  const { history } = usePath(language, nativeLanguage)

  if (!history.data || history.data.length === 0) return null

  return (
    <section className="space-y-2.5">
      <h2 className="text-base font-semibold">{t("history")}</h2>
      <ul className="space-y-2">
        {history.data.map((item) => (
          <li
            key={item.id}
            className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-4"
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
