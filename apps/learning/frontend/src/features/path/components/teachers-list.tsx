"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { IconPlus, IconTrash } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { useTeachers } from "../hooks"
import { languages } from "../languages"

export type TeacherPair = { language: string; translationLanguage: string }

/**
 * The learner's teachers: one per language they study, each with its own
 * program and level, side by side like the topics of the notebook. Opening one
 * hands its pair to the host, which shows that teacher's program.
 */
export function TeachersList({ onOpen }: { onOpen: (pair: TeacherPair) => void }) {
  const t = useTranslations("path")
  const locale = useLocale()
  const names = new Intl.DisplayNames([locale], { type: "language" })
  const { teachers, create, remove } = useTeachers()
  const [adding, setAdding] = useState(false)
  const [confirming, setConfirming] = useState<string | null>(null)
  const [language, setLanguage] = useState("")
  const [native, setNative] = useState(locale.split("-")[0] === "en" ? "en" : "ru")

  const have = new Set(teachers.data?.map((teacher) => teacher.language))
  const choices = languages.filter((code) => !have.has(code) && code !== native)
  const picked = language && choices.includes(language) ? language : (choices[0] ?? "")
  const empty = teachers.isSuccess && teachers.data.length === 0
  const form = adding || empty

  return (
    <div className="anim-fade space-y-4 lg:mx-auto lg:w-full lg:max-w-[704px]">
      <header className="space-y-1">
        <h1 className="text-[22px] font-semibold tracking-[-0.01em] lg:text-2xl">
          {t("teachersTitle")}
        </h1>
        <p className="text-muted-foreground text-xs leading-relaxed lg:text-sm">
          {t("teachersSubtitle")}
        </p>
      </header>

      {teachers.isPending && (
        <div className="space-y-3">
          <Skeleton className="h-20 w-full rounded-lg" />
          <Skeleton className="h-20 w-full rounded-lg" />
        </div>
      )}

      {teachers.isError && (
        <p role="alert" className="text-destructive text-sm">
          {teachers.error.message}
        </p>
      )}

      {teachers.data && teachers.data.length > 0 && (
        <ul className="space-y-2.5">
          {teachers.data.map((teacher) => {
            const path = teacher.path
            const percent =
              path && path.topicsTotal > 0
                ? Math.round((path.topicsDone / path.topicsTotal) * 100)
                : 0

            return (
              <li key={teacher.id} className="space-y-2">
                <div className="flex items-stretch gap-2">
                <button
                  type="button"
                  onClick={() =>
                    onOpen({
                      language: teacher.language,
                      translationLanguage: teacher.translationLanguage,
                    })
                  }
                  className="bg-background border-border hover:bg-muted/40 block min-w-0 flex-1 cursor-pointer space-y-2.5 rounded-lg border p-4 text-left transition-colors"
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-[15px] font-semibold">
                        {names.of(teacher.language) ?? teacher.language}
                      </p>
                      <p className="text-muted-foreground truncate text-xs">
                        {t("explainedIn", {
                          language: names.of(teacher.explainLanguage) ?? teacher.explainLanguage,
                        })}
                      </p>
                    </div>
                    {path ? (
                      <span className="bg-success/15 text-success shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold">
                        {path.level}
                      </span>
                    ) : (
                      <span className="text-muted-foreground shrink-0 text-xs">
                        {t("teacherNoProgram")}
                      </span>
                    )}
                  </div>
                  {path && (
                    <div className="space-y-1.5">
                      <div
                        role="progressbar"
                        aria-valuemin={0}
                        aria-valuemax={path.topicsTotal}
                        aria-valuenow={path.topicsDone}
                        className="bg-border h-1.5 overflow-hidden rounded-full"
                      >
                        <div
                          className="bg-success h-full rounded-full transition-[width] duration-300"
                          style={{ width: `${percent}%` }}
                        />
                      </div>
                      <p className="text-muted-foreground text-xs">
                        {t("teacherProgress", { done: path.topicsDone, total: path.topicsTotal })}
                      </p>
                    </div>
                  )}
                </button>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label={t("deleteTeacher")}
                  title={t("deleteTeacher")}
                  onClick={() => setConfirming(confirming === teacher.id ? null : teacher.id)}
                  className="text-muted-foreground hover:text-destructive h-auto w-11 shrink-0"
                >
                  <IconTrash size={16} />
                </Button>
                </div>
                {confirming === teacher.id && (
                  <div
                    role="alertdialog"
                    className="border-destructive/40 bg-background space-y-2.5 rounded-lg border p-3.5"
                  >
                    <p className="text-sm leading-relaxed">
                      {t(teacher.path ? "deleteTeacherWithProgram" : "deleteTeacherConfirm", {
                        language: names.of(teacher.language) ?? teacher.language,
                      })}
                    </p>
                    {remove.isError && (
                      <p role="alert" className="text-destructive text-xs">
                        {remove.error.message}
                      </p>
                    )}
                    <div className="flex gap-2">
                      <Button
                        variant="destructive"
                        size="lg"
                        disabled={remove.isPending}
                        onClick={() =>
                          remove.mutate(teacher.id, { onSuccess: () => setConfirming(null) })
                        }
                        className="px-4 text-[13px]"
                      >
                        {t(remove.isPending ? "deleting" : "deleteConfirm")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="lg"
                        onClick={() => setConfirming(null)}
                        className="px-4 text-[13px]"
                      >
                        {t("cancel")}
                      </Button>
                    </div>
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}

      {teachers.isSuccess && !form && choices.length > 0 && (
        <Button variant="outline" size="lg" onClick={() => setAdding(true)} className="w-full text-[13px]">
          <IconPlus size={16} />
          {t("addTeacher")}
        </Button>
      )}

      {teachers.isSuccess && form && (
        <section className="bg-background border-border space-y-3 rounded-lg border p-5">
          <p className="text-sm leading-relaxed">{t("addTeacherHint")}</p>
          <div className="grid gap-3 sm:grid-cols-2">
            {(
              [
                [picked, setLanguage, "language", choices],
                [native, setNative, "nativeLanguage", languages.filter((code) => code !== picked)],
              ] as const
            ).map(([value, setValue, name, options]) => (
              <label key={name} className="block space-y-1.5 text-[13px] font-medium">
                <span>{t(name)}</span>
                <Select value={value} onValueChange={setValue}>
                  <SelectTrigger className="bg-background h-10 w-full px-3 text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {options.map((code) => (
                      <SelectItem key={code} value={code}>
                        {names.of(code) ?? code}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>
            ))}
          </div>
          {create.isError && (
            <p role="alert" className="text-destructive text-sm">
              {create.error.message}
            </p>
          )}
          <div className="flex gap-3">
            <Button
              size="lg"
              disabled={create.isPending || !picked}
              onClick={() =>
                create.mutate(
                  { language: picked, translationLanguage: native },
                  {
                    onSuccess: (teacher) => {
                      setAdding(false)
                      onOpen({
                        language: teacher.language,
                        translationLanguage: teacher.translationLanguage,
                      })
                    },
                  },
                )
              }
              className="px-4 text-[13px]"
            >
              {t(create.isPending ? "adding" : "add")}
            </Button>
            {!empty && (
              <Button variant="ghost" size="lg" onClick={() => setAdding(false)} className="px-4 text-[13px]">
                {t("cancel")}
              </Button>
            )}
          </div>
        </section>
      )}
    </div>
  )
}
