"use client"

import { useId } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Switch } from "@/components/ui/switch"
import { useTeachers } from "../hooks"

/**
 * The learner's choice of the language a teacher explains in: their own, or
 * the one they are learning. It decides what the rule book, the program, the
 * exercises and the explanations of mistakes are written in, so a learner who
 * can already read the language is not given a translation to read first.
 */
export function ExplainSwitch({
  language,
  className,
}: {
  language?: string
  className?: string
}) {
  const t = useTranslations("path")
  const locale = useLocale()
  const id = useId()
  const { teachers, update } = useTeachers()
  const teacher = language
    ? teachers.data?.find((item) => item.language === language)
    : undefined

  if (!teacher) return null

  const names = new Intl.DisplayNames([locale], { type: "language" })
  const learned = names.of(teacher.language) ?? teacher.language
  const own = names.of(teacher.translationLanguage) ?? teacher.translationLanguage
  // The switch moves as it is pressed; it goes back if the change is refused.
  const on = update.isPending ? update.variables.explainInTarget : teacher.explainInTarget

  return (
    <div className={className}>
      <div className="bg-background border-border flex items-start justify-between gap-4 rounded-lg border px-4 py-3.5">
        <label htmlFor={id} className="min-w-0 flex-1 cursor-pointer space-y-1">
          <span className="block text-sm font-medium">{t("explainTarget")}</span>
          <span className="text-muted-foreground block text-xs leading-relaxed">
            {t(on ? "explainTargetOn" : "explainTargetOff", { learned, own })}
          </span>
        </label>
        <Switch
          id={id}
          checked={on}
          disabled={update.isPending}
          onCheckedChange={(explainInTarget) =>
            update.mutate({ id: teacher.id, explainInTarget })
          }
          className="mt-0.5"
        />
      </div>
      {update.isError && (
        <p role="alert" className="text-destructive mt-2 text-xs">
          {update.error.message}
        </p>
      )}
    </div>
  )
}
