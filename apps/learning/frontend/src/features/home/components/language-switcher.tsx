"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import { IconCheck, IconPlus, IconSelector } from "@tabler/icons-react"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { PathTeacher } from "@/features/path/schemas"
import { cn } from "@/lib/utils"

import { useActiveTeacher, useTeacherStatus } from "../setup"
import { CodeBadge } from "./parts"

/** The name of a language in the learner's own interface language. */
export function useLanguageName() {
  const locale = useLocale()
  const names = new Intl.DisplayNames([locale], { type: "language" })

  return (code: string) => {
    const name = names.of(code) ?? code

    return name.charAt(0).toLocaleUpperCase(locale) + name.slice(1)
  }
}

/** What a language is waiting for, as a short line under its name. */
export function StatusLine({ teacher }: { teacher: PathTeacher }) {
  const t = useTranslations("home.status")
  const key = useTeacherStatus(teacher)

  return <>{t(key)}</>
}

/**
 * The language on screen, with the others one tap away. The learner's teachers
 * are separate profiles, each with its own program, so this is also where they
 * add a language or go to the list that manages them.
 */
export function LanguageSwitcher({ className }: { className?: string }) {
  const t = useTranslations("home")
  const name = useLanguageName()
  const router = useRouter()
  const pathname = usePathname()
  const { teachers, list, active, setActive } = useActiveTeacher()

  if (teachers.isPending) {
    return (
      <div
        aria-hidden="true"
        className={cn("bg-muted h-11 animate-pulse rounded-lg", className)}
      />
    )
  }

  if (!active) {
    return (
      <Link
        href="/learning/start"
        className={cn(
          "border-input bg-card hover:bg-muted flex min-h-11 items-center gap-2 rounded-lg border border-dashed px-2 text-sm font-medium",
          className,
        )}
      >
        <IconPlus size={16} aria-hidden="true" />
        {t("addLanguage")}
      </Link>
    )
  }

  const pick = (language: string) => {
    setActive(language)
    if (pathname.startsWith("/learning/path/rules")) {
      router.push(`/learning/knowledge?language=${encodeURIComponent(language)}`)
    } else if (["/learning/path", "/learning/knowledge", "/learning/practice", "/learning/progress"].includes(pathname)) {
      router.push(`${pathname}?language=${encodeURIComponent(language)}`)
    } else if (pathname === "/learning/languages" || pathname === "/learning/start") {
      router.push("/learning/overview")
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={t("switchProfile")}
          className={cn(
            "border-border bg-card hover:bg-muted focus-visible:ring-ring/60 flex min-h-11 w-full cursor-pointer items-center gap-2 rounded-lg border p-2 text-left text-sm outline-none focus-visible:ring-2",
            className,
          )}
        >
          <CodeBadge code={active.language} />
          <span className="min-w-0 flex-1 leading-tight">
            <span className="block truncate font-medium">{name(active.language)}</span>
            <span className="language-switcher-detail text-muted-foreground block text-xs">{t("aiTeacher")}</span>
          </span>
          <IconSelector size={16} aria-hidden="true" className="text-muted-foreground shrink-0" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[max(15rem,var(--radix-dropdown-menu-trigger-width))]">
        <DropdownMenuLabel className="text-muted-foreground text-[11px] font-medium tracking-[0.04em] uppercase">
          {t("profiles")}
        </DropdownMenuLabel>
        {list.map((teacher) => {
          const on = teacher.language === active.language

          return (
            <DropdownMenuItem
              key={teacher.id}
              onSelect={() => pick(teacher.language)}
              className={cn("min-h-11 gap-2", on && "bg-muted")}
            >
              <span className="min-w-0 flex-1 leading-tight">
                <span className="block truncate font-medium">{name(teacher.language)}</span>
                <span className="text-muted-foreground block text-xs">
                  <StatusLine teacher={teacher} />
                </span>
              </span>
              {on && (
                <span className="flex items-center gap-1 text-xs font-medium">
                  <IconCheck size={14} aria-hidden="true" />
                  {t("active")}
                </span>
              )}
            </DropdownMenuItem>
          )
        })}
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild className="min-h-10">
          <Link href="/learning/start">+ {t("addLanguage")}</Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild className="min-h-10">
          <Link href="/learning/languages">{t("manageLanguages")}</Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
