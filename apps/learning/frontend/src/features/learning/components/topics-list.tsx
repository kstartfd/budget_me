"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconChevronRight, IconPlus } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useProvideAddAction } from "@/components/add-action"
import { languageBadge, useLanguageName } from "../language"
import type { Topic } from "../schemas"

export function TopicsList({
  topics,
  onNewTopic,
}: {
  topics: Topic[]
  onNewTopic: () => void
}) {
  const t = useTranslations("learning")
  const languageName = useLanguageName()
  useProvideAddAction(onNewTopic)

  return (
    <div className="anim-fade space-y-[18px] lg:mx-auto lg:max-w-[760px] lg:space-y-[22px] lg:pt-5">
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-[26px] font-bold tracking-[-0.02em] break-words lg:text-[28px]">
            {t("title")}
          </h1>
          <p className="text-muted-foreground mt-1.5 text-[13px] leading-relaxed lg:text-sm">
            {t("subtitle")}
          </p>
        </div>
      </header>

      <div className="grid gap-2.5 lg:grid-cols-3 lg:gap-3.5">
        {topics.map((item) => (
          <Link
            key={item.id}
            href={`/learning/${item.id}`}
            className="bg-card border-border hover:border-primary/30 flex min-w-0 items-center gap-3.5 rounded-[18px] border p-4 shadow-[var(--shadow-card)] transition-colors lg:p-[18px]"
          >
            <span className="bg-[color:var(--primary-wash)] text-primary flex size-11 shrink-0 items-center justify-center rounded-full text-sm font-bold">
              {languageBadge(item.language)}
            </span>
            <div className="min-w-0 flex-1">
              <h2 className="line-clamp-2 text-base leading-snug font-semibold break-words">
                {item.name}
              </h2>
              <p className="text-muted-foreground mt-0.5 text-xs leading-relaxed break-words">
                {languageName(item.language)} →{" "}
                {languageName(item.translationLanguage)}
              </p>
            </div>
            <IconChevronRight size={17} className="text-faint shrink-0" />
          </Link>
        ))}
        <Button
          variant="outline"
          onClick={onNewTopic}
          className="text-primary border-primary/30 h-auto min-h-14 rounded-[18px] border-dashed bg-transparent p-[18px] text-sm font-semibold"
        >
          <IconPlus size={18} />
          {t("newTopic")}
        </Button>
      </div>

      {topics.length === 0 && (
        <p className="text-muted-foreground max-w-lg text-sm leading-relaxed">
          {t("topicsEmpty")}
        </p>
      )}
    </div>
  )
}
