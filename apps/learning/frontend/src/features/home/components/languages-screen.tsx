"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { useTeachers } from "@/features/path/hooks"
import type { PathTeacher } from "@/features/path/schemas"
import { cn } from "@/lib/utils"

import { useProfile } from "../profile"
import { useActiveTeacher } from "../setup"
import { StatusLine, useLanguageName } from "./language-switcher"
import { Pill, card } from "./parts"

/**
 * Every language the learner studies, side by side: switch to one, open its
 * teacher settings, or remove it. Each language has its own settings, program,
 * progress and history, so removing one touches none of the others.
 */
export function LanguagesScreen() {
  const t = useTranslations("home")
  const name = useLanguageName()
  const router = useRouter()
  const { remove } = useTeachers()
  const { teachers, list, active, setActive } = useActiveTeacher()
  const [removing, setRemoving] = useState<PathTeacher | null>(null)
  const [typed, setTyped] = useState("")

  const close = () => {
    setRemoving(null)
    setTyped("")
  }

  return (
    <div className="anim-fade space-y-6">
      <header>
        <p className="text-muted-foreground mb-1 text-xs">{t("crumb", { page: t("manageLanguages") })}</p>
        <h1 className="text-2xl font-semibold tracking-[-0.01em]">{t("manageLanguages")}</h1>
        <p className="text-muted-foreground mt-0.5 text-sm">{t("manageSub")}</p>
      </header>

      <section className={cn(card, "max-w-[960px]")}>
        <div className="border-border flex flex-wrap items-center justify-between gap-2 border-b px-5 py-4">
          <div>
            <h2 className="text-base font-semibold">{t("profiles")}</h2>
            <p className="text-muted-foreground text-xs">{t("manageHint")}</p>
          </div>
          <Button asChild variant="outline" size="lg" className="h-10 px-3.5 text-sm">
            <Link href="/learning/start">+ {t("addLanguage")}</Link>
          </Button>
        </div>

        {teachers.isPending && (
          <div className="space-y-3 p-5">
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
          </div>
        )}

        {teachers.isError && (
          <p role="alert" className="text-destructive p-5 text-sm">
            {teachers.error.message}
          </p>
        )}

        {teachers.isSuccess && list.length === 0 && (
          <p className="text-muted-foreground p-5 text-sm">{t("manageEmpty")}</p>
        )}

        <ul>
          {list.map((teacher) => {
            const on = teacher.language === active?.language

            return (
              <li
                key={teacher.id}
                className="border-border flex flex-wrap items-center gap-3 border-t px-5 py-4 first:border-t-0"
              >
                <div className="min-w-60 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-semibold">{name(teacher.language)}</span>
                    {on && (
                      <Pill tone="neutral" className="py-px">
                        ✓ {t("active")}
                      </Pill>
                    )}
                  </div>
                  <p className="text-muted-foreground text-[13px]">
                    <StatusLine teacher={teacher} /> · <Settings teacher={teacher} />
                  </p>
                </div>
                <div className="ml-auto flex flex-wrap justify-end gap-2">
                  {!on && (
                    <Button
                      variant="outline"
                      size="lg"
                      onClick={() => {
                        setActive(teacher.language)
                        router.push("/learning/overview")
                      }}
                      className="h-9 px-3 text-[13px]"
                    >
                      {t("switchTo")}
                    </Button>
                  )}
                  <Button asChild variant="outline" size="lg" className="h-9 px-3 text-[13px]">
                    <Link href={`/learning/path?language=${teacher.language}`}>{t("teacherSettings")}</Link>
                  </Button>
                  <Button
                    variant="outline"
                    size="lg"
                    onClick={() => setRemoving(teacher)}
                    className="text-warn h-9 px-3 text-[13px]"
                  >
                    {t("remove")}
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      </section>

      <Dialog open={!!removing} onOpenChange={(open) => !open && close()}>
        <DialogContent role="alertdialog" className="gap-3 rounded-xl p-6 text-sm sm:max-w-[440px]">
          {removing && (
            <>
              <DialogHeader>
                <DialogTitle className="text-lg font-semibold">
                  {t("removeTitle", { language: name(removing.language) })}
                </DialogTitle>
                <DialogDescription className="text-muted-foreground text-sm leading-relaxed">
                  {t("removeText", { language: name(removing.language) })}
                </DialogDescription>
              </DialogHeader>
              <Input
                aria-label={t("removeTypeLabel")}
                value={typed}
                onChange={(event) => setTyped(event.target.value)}
                autoComplete="off"
                className="bg-background h-10 text-sm md:text-sm"
              />
              {remove.isError && (
                <p role="alert" className="text-destructive text-[13px]">
                  {remove.error.message}
                </p>
              )}
              <DialogFooter className="flex-row justify-end gap-2">
                <Button variant="outline" size="lg" onClick={close} className="h-10 px-3.5 text-sm">
                  {t("cancel")}
                </Button>
                <Button
                  variant="destructive"
                  size="lg"
                  disabled={
                    remove.isPending ||
                    typed.trim().toLocaleLowerCase() !== name(removing.language).toLocaleLowerCase()
                  }
                  onClick={() =>
                    remove.mutate(removing.id, {
                      onSuccess: close,
                    })
                  }
                  className="h-10 px-3.5 text-sm"
                >
                  {t(remove.isPending ? "removing" : "removeConfirm")}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

/** "Explanations in English · 5h per week", from what the teacher and its profile know. */
function Settings({ teacher }: { teacher: PathTeacher }) {
  const t = useTranslations("home")
  const name = useLanguageName()
  const [profile] = useProfile(teacher.language)
  const hours = profile?.weekly

  return (
    <>
      {t("explainsIn", { language: name(teacher.explainLanguage) })}
      {hours ? ` · ${t("perWeek", { hours })}` : ""}
    </>
  )
}
