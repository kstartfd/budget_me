"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconChevronRight, IconKey } from "@tabler/icons-react"

/** The way to the tokens an extension saves words with; it lives in settings. */
export function ExtensionSection() {
  const t = useTranslations("learning")

  return (
    <section className="flex flex-col gap-[9px]">
      <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
        {t("tokens")}
      </span>

      <Link
        href="/learning/tokens"
        className="bg-card border-border hover:border-primary/30 flex items-center gap-3.5 rounded-[18px] border p-4 shadow-[var(--shadow-card)] transition-colors"
      >
        <span className="bg-[color:var(--primary-wash)] text-primary flex size-10 shrink-0 items-center justify-center rounded-full">
          <IconKey size={18} />
        </span>
        <span className="min-w-0 flex-1 text-sm leading-snug font-semibold break-words">
          {t("extensionHint")}
        </span>
        <IconChevronRight size={17} className="text-faint shrink-0" />
      </Link>
    </section>
  )
}
