"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconChevronRight, IconKey } from "@tabler/icons-react"

import { SectionKicker, sectionCard } from "@/components/section"
import { cn } from "@/lib/utils"

/** The way to the tokens an extension saves words with; it lives in settings. */
export function ExtensionSection() {
  const t = useTranslations("learning")

  return (
    <section className="flex flex-col gap-3">
      <SectionKicker>{t("tokens")}</SectionKicker>

      <Link
        href="/learning/tokens"
        className={cn(
          sectionCard,
          "hover:border-primary/30 flex items-center gap-3.5 p-4 transition-colors",
        )}
      >
        <span className="bg-muted text-foreground/75 flex size-[38px] shrink-0 items-center justify-center rounded-[9px]">
          <IconKey size={18} />
        </span>
        <span className="min-w-0 flex-1 text-sm leading-snug font-semibold break-words">
          {t("extensionHint")}
        </span>
        <IconChevronRight
          size={17}
          className="text-muted-foreground shrink-0"
          aria-hidden="true"
        />
      </Link>
    </section>
  )
}
