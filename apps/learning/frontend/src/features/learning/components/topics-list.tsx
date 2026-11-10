"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconChevronRight,
  IconPlus,
  IconSearch,
  IconSettings,
} from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { languageBadge, useLanguageName } from "../language"
import type { Topic } from "../schemas"

const columns =
  "grid grid-cols-[minmax(0,1fr)_28px_16px] items-center gap-3 px-4 lg:grid-cols-[minmax(0,1fr)_220px_28px_16px]"

export function TopicsList({
  topics,
  onNewTopic,
  onEditTopic,
}: {
  topics: Topic[]
  onNewTopic: () => void
  onEditTopic: (topic: Topic) => void
}) {
  const t = useTranslations("learning")
  const languageName = useLanguageName()
  const [query, setQuery] = useState("")
  const needle = query.trim().toLowerCase()
  const shown = needle
    ? topics.filter((item) => item.name.toLowerCase().includes(needle))
    : topics

  return (
    <div className="anim-fade space-y-4 lg:space-y-5">
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-[22px] font-semibold tracking-[-0.01em] break-words lg:text-2xl">
            {t("title")}
          </h1>
          <p className="text-muted-foreground mt-1 text-xs leading-relaxed lg:text-sm">
            {t("subtitle")}
          </p>
        </div>
        <Button
          size="lg"
          aria-label={t("newTopic")}
          onClick={onNewTopic}
          className="size-9 shrink-0 p-0 lg:h-9 lg:w-auto lg:px-4 lg:text-sm"
        >
          <IconPlus size={14} />
          <span className="hidden lg:inline">{t("newTopic")}</span>
        </Button>
      </header>

      {topics.length > 0 && (
        <div className="relative lg:max-w-80">
          <IconSearch
            size={15}
            className="text-faint pointer-events-none absolute top-1/2 left-3 -translate-y-1/2"
          />
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("searchTopics")}
            aria-label={t("searchTopics")}
            className="h-9 pl-9 text-sm md:text-sm"
          />
        </div>
      )}

      {shown.length > 0 && (
        <div className="border-border bg-background overflow-hidden rounded-lg border">
          <div
            className={`${columns} border-border text-muted-foreground hidden h-10 border-b text-[13px] font-medium lg:grid`}
          >
            <span>{t("colTopic")}</span>
            <span>{t("colLanguages")}</span>
            <span />
            <span />
          </div>
          {shown.map((item) => (
            <div
              key={item.id}
              className={`${columns} border-border hover:bg-muted/50 relative min-h-14 border-b py-3 transition-colors last:border-b-0`}
            >
              <div className="flex min-w-0 items-center gap-3">
                <span className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-md text-xs font-semibold">
                  {languageBadge(item.language)}
                </span>
                <div className="min-w-0">
                  <h2 className="line-clamp-2 text-sm leading-snug font-medium break-words">
                    <Link
                      href={`/learning/${item.id}`}
                      className="after:absolute after:inset-0"
                    >
                      {item.name}
                    </Link>
                  </h2>
                  <p className="text-muted-foreground mt-0.5 text-xs break-words lg:hidden">
                    {languageName(item.language)} →{" "}
                    {languageName(item.translationLanguage)}
                  </p>
                </div>
              </div>
              <p className="text-muted-foreground hidden text-sm lg:block">
                {languageName(item.language)} →{" "}
                {languageName(item.translationLanguage)}
              </p>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("editTopic")}
                onClick={() => onEditTopic(item)}
                className="text-muted-foreground relative z-10"
              >
                <IconSettings size={15} />
              </Button>
              <IconChevronRight size={14} className="text-faint" />
            </div>
          ))}
        </div>
      )}

      {topics.length === 0 && (
        <p className="text-muted-foreground max-w-lg text-sm leading-relaxed">
          {t("topicsEmpty")}
        </p>
      )}
      {topics.length > 0 && shown.length === 0 && (
        <p className="text-muted-foreground text-sm">{t("noMatches")}</p>
      )}
    </div>
  )
}
