"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import {
  IconArrowLeft,
  IconChevronRight,
  IconKey,
  IconLanguage,
  IconPlus,
  IconSearch,
  IconSettings,
  IconTrash,
} from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { useProvideAddAction } from "@/components/add-action"
import { useIsDesktop } from "@/hooks/use-is-desktop"
import { cn } from "@/lib/utils"
import { useEntries, useEntry, useLearning } from "../hooks"
import type { Entry, Topic } from "../schemas"
import { EntryDetail } from "./entry-detail"
import { EntryForm } from "./entry-form"
import { LearningModal } from "./learning-modal"
import { TopicForm } from "./topic-form"

export function LearningScreen({ topicId = "" }: { topicId?: string }) {
  const t = useTranslations("learning")
  const locale = useLocale()
  const router = useRouter()
  const desktop = useIsDesktop()
  const { topics, deleteTopic } = useLearning()
  const [topicForm, setTopicForm] = useState<Topic | "new" | null>(null)
  const [entryForm, setEntryForm] = useState<Entry | "new" | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [kind, setKind] = useState<"all" | "word" | "phrase">("all")
  const [status, setStatus] = useState<"all" | "inbox" | "learning" | "known">(
    "all",
  )
  const [confirmDelete, setConfirmDelete] = useState(false)
  const entries = useEntries(topicId, { kind, status, search })
  const topic = topics.data?.find((item) => item.id === topicId)
  const rows = entries.data?.pages.flat() ?? []
  const selectedEntry = useEntry(
    selectedId,
    rows.find((entry) => entry.id === selectedId),
  )
  const selected = selectedEntry.isError ? undefined : selectedEntry.data
  useProvideAddAction(() => {
    if (topicId) setEntryForm("new")
    else setTopicForm("new")
  })

  function languageName(code: string) {
    try {
      return (
        new Intl.DisplayNames([locale], { type: "language" }).of(code) ?? code
      )
    } catch {
      return code
    }
  }
  if (topics.isPending)
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-36" />
        <Skeleton className="h-40 w-full rounded-[22px]" />
      </div>
    )
  if (topics.isError)
    return (
      <p
        role="alert"
        className="text-muted-foreground py-10 text-center text-sm"
      >
        {t("loadError")}
      </p>
    )
  if (topicId && !topic)
    return (
      <div className="space-y-3">
        <p>{t("topicMissing")}</p>
        <Link href="/learning" className="text-primary text-sm">
          {t("back")}
        </Link>
      </div>
    )

  return (
    <div className="anim-fade space-y-[18px]">
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          {topic && (
            <Link
              href="/learning"
              className="text-muted-foreground mb-2 inline-flex items-center gap-1 text-xs"
            >
              <IconArrowLeft size={14} />
              {t("title")}
            </Link>
          )}
          <h1 className="text-[22px] font-bold tracking-[-0.02em] break-words">
            {topic?.name ?? t("title")}
          </h1>
          <p className="text-muted-foreground mt-1 text-xs leading-relaxed">
            {topic
              ? `${languageName(topic.language)} → ${languageName(topic.translationLanguage)}`
              : t("subtitle")}
          </p>
        </div>
        <div className="flex shrink-0 gap-2 pt-1">
          {topic ? (
            <Button
              variant="secondary"
              size="icon"
              className="border-border rounded-full"
              aria-label={t("editTopic")}
              onClick={() => setTopicForm(topic)}
            >
              <IconSettings size={18} />
            </Button>
          ) : (
            <Link
              href="/learning/tokens"
              className="bg-secondary border-border text-muted-foreground flex size-9 items-center justify-center rounded-full border"
              aria-label={t("tokens")}
            >
              <IconKey size={18} />
            </Link>
          )}
        </div>
      </header>

      {!topic ? (
        <>
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {topics.data?.map((item) => (
              <Link
                key={item.id}
                href={`/learning/${item.id}`}
                className="bg-card border-border hover:border-primary/30 flex min-w-0 items-start gap-3 rounded-[22px] border p-5 shadow-[var(--shadow-card)] transition-colors"
              >
                <span className="bg-[color:var(--primary-wash)] text-primary flex size-11 shrink-0 items-center justify-center rounded-2xl">
                  <IconLanguage size={24} />
                </span>
                <div className="min-w-0 flex-1">
                  <h2 className="line-clamp-2 text-base leading-snug font-semibold break-words">
                    {item.name}
                  </h2>
                  <p className="text-faint mt-1 text-xs leading-relaxed break-words">
                    {languageName(item.language)} →{" "}
                    {languageName(item.translationLanguage)}
                  </p>
                </div>
                <IconChevronRight
                  size={17}
                  className="text-faint mt-3 shrink-0"
                />
              </Link>
            ))}
            <Button
              variant="outline"
              onClick={() => setTopicForm("new")}
              className="text-primary h-auto min-h-24 rounded-[22px] border-dashed bg-transparent p-5 text-sm"
            >
              <IconPlus size={19} />
              {t("newTopic")}
            </Button>
          </div>
          {topics.data?.length === 0 && (
            <p className="text-muted-foreground max-w-lg text-sm leading-relaxed">
              {t("topicsEmpty")}
            </p>
          )}
          <Link
            href="/learning/tokens"
            className="text-muted-foreground inline-flex items-center gap-2 text-xs"
          >
            <IconKey size={15} />
            {t("extensionHint")}
            <IconChevronRight size={14} />
          </Link>
        </>
      ) : (
        <>
          <div
            className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap"
            role="group"
            aria-label={t("status")}
          >
            {(["all", "inbox", "learning", "known"] as const).map((value) => (
              <Button
                key={value}
                variant="secondary"
                size="sm"
                aria-pressed={status === value}
                onClick={() => setStatus(value)}
                className={cn(
                  "h-9 min-w-0 rounded-full px-2 text-xs sm:px-4",
                  status === value
                    ? "bg-primary/15 text-primary ring-primary/25 ring-1"
                    : "border-border",
                )}
              >
                {t(value)}
              </Button>
            ))}
          </div>
          <div className="grid min-w-0 gap-3 lg:grid-cols-[auto_minmax(0,1fr)_auto] lg:items-center">
            <div
              className="bg-secondary border-border flex min-w-0 gap-1 rounded-xl border p-1"
              role="group"
              aria-label={t("kind")}
            >
              {(["all", "word", "phrase"] as const).map((value) => (
                <Button
                  key={value}
                  variant="ghost"
                  size="sm"
                  onClick={() => setKind(value)}
                  aria-pressed={kind === value}
                  className={cn(
                    "h-8 min-w-0 flex-1 rounded-lg px-2 text-xs lg:flex-none lg:px-3",
                    kind === value && "bg-card text-foreground shadow-sm",
                  )}
                >
                  {t(
                    value === "all"
                      ? "allKinds"
                      : value === "word"
                        ? "words"
                        : "phrases",
                  )}
                </Button>
              ))}
            </div>
            <div className="relative min-w-0">
              <IconSearch
                size={16}
                className="text-faint absolute top-3 left-3"
              />
              <Input
                aria-label={t("search")}
                placeholder={t("search")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="bg-card border-border focus-visible:border-primary h-10 rounded-xl pl-9 text-sm focus-visible:ring-0"
              />
            </div>
            <Button
              onClick={() => setEntryForm("new")}
              className="text-primary-foreground hidden h-10 rounded-xl px-4 text-sm font-semibold lg:inline-flex"
            >
              <IconPlus size={17} />
              {t("newEntry")}
            </Button>
          </div>
          <div className="grid items-stretch gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.85fr)]">
            <section
              className="bg-card border-border min-w-0 overflow-hidden rounded-[22px] border shadow-[var(--shadow-card)]"
              aria-label={t("entries")}
            >
              {entries.isPending ? (
                <div className="space-y-3 p-5">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : entries.isError ? (
                <p role="alert" className="text-muted-foreground p-6 text-sm">
                  {t("loadError")}
                </p>
              ) : rows.length === 0 ? (
                <div className="p-6">
                  <p className="text-sm font-semibold">
                    {t(
                      search || kind !== "all" || status !== "all"
                        ? "noMatches"
                        : "entriesEmpty",
                    )}
                  </p>
                  <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
                    {t(
                      search || kind !== "all" || status !== "all"
                        ? "noMatchesHint"
                        : "entriesEmptyHint",
                    )}
                  </p>
                  {rows.length === 0 && (
                    <Button
                      variant="secondary"
                      className="text-primary mt-4 h-10 rounded-xl px-4 text-sm font-semibold"
                      onClick={() => setEntryForm("new")}
                    >
                      <IconPlus size={16} />
                      {t("newEntry")}
                    </Button>
                  )}
                </div>
              ) : (
                <div className="divide-rowline divide-y">
                  {rows.map((entry) => (
                    <button
                      key={entry.id}
                      type="button"
                      onClick={() => setSelectedId(entry.id)}
                      className={cn(
                        "hover:bg-secondary/60 flex w-full min-w-0 gap-3 p-4 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary",
                        selectedId === entry.id && "bg-primary/5",
                      )}
                    >
                      <span
                        className={cn(
                          "mt-2 size-2 shrink-0 rounded-full",
                          entry.status === "known"
                            ? "bg-success"
                            : entry.status === "learning"
                              ? "bg-primary"
                              : "bg-warn",
                        )}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="line-clamp-2 text-[15px] leading-snug font-semibold break-words">
                          {entry.text}
                        </div>
                        <p className="text-muted-foreground mt-1 line-clamp-2 text-xs leading-relaxed break-words">
                          {entry.translation || t("unfilled")}
                        </p>
                        <span className="text-faint mt-2 block text-[10px]">
                          {t(entry.kind)} · {t(entry.status)}
                        </span>
                        {entry.source && (
                          <span className="text-faint mt-1 block truncate text-[10px]">
                            {entry.source}
                          </span>
                        )}
                      </div>
                      <IconChevronRight
                        className="text-faint mt-1 shrink-0"
                        size={16}
                      />
                    </button>
                  ))}
                </div>
              )}
              {entries.hasNextPage && (
                <div className="border-border border-t p-3">
                  <Button
                    variant="ghost"
                    className="w-full text-sm"
                    disabled={entries.isFetchingNextPage}
                    onClick={() => entries.fetchNextPage()}
                  >
                    {t("loadMore")}
                  </Button>
                </div>
              )}
            </section>
            <aside className="bg-card border-border sticky top-6 hidden min-w-0 self-stretch rounded-[22px] border p-5 shadow-[var(--shadow-card)] lg:flex lg:flex-col">
              {desktop && selected ? (
                <EntryDetail
                  key={selected.id}
                  entry={selected}
                  onEdit={() => setEntryForm(selected)}
                  onDelete={() => setSelectedId(null)}
                />
              ) : selectedId ? (
                selectedEntry.isError ? (
                  <p role="alert" className="text-muted-foreground text-sm">
                    {t("loadError")}
                  </p>
                ) : (
                  <Skeleton className="h-40 w-full" />
                )
              ) : (
                <div className="text-muted-foreground m-auto max-w-[220px] py-12 text-center text-sm leading-relaxed">
                  <IconLanguage size={30} className="text-faint mx-auto mb-3" />
                  {t("selectEntry")}
                </div>
              )}
            </aside>
          </div>
          <div className="flex items-center justify-between gap-3 pt-1">
            <Link
              href="/learning/tokens"
              className="text-muted-foreground inline-flex items-center gap-2 text-xs"
            >
              <IconKey size={15} />
              {t("tokens")}
            </Link>
            <Button
              variant="ghost"
              className="text-faint text-xs"
              onClick={() => setConfirmDelete(true)}
            >
              <IconTrash size={14} />
              {t("deleteTopic")}
            </Button>
          </div>
        </>
      )}

      {topicForm && (
        <TopicForm
          topic={topicForm === "new" ? undefined : topicForm}
          onClose={() => setTopicForm(null)}
        />
      )}
      {entryForm && topic && (
        <EntryForm
          entry={entryForm === "new" ? undefined : entryForm}
          topicId={topic.id}
          kind={kind === "word" ? "word" : "phrase"}
          onClose={() => setEntryForm(null)}
        />
      )}
      {selectedId && !desktop && !entryForm && (
        <LearningModal
          title={t("entryDetails")}
          onClose={() => setSelectedId(null)}
        >
          {selected ? (
            <EntryDetail
              key={selected.id}
              entry={selected}
              onEdit={() => setEntryForm(selected)}
              onDelete={() => setSelectedId(null)}
            />
          ) : selectedEntry.isError ? (
            <p role="alert" className="text-muted-foreground text-sm">
              {t("loadError")}
            </p>
          ) : (
            <Skeleton className="h-40 w-full" />
          )}
        </LearningModal>
      )}
      {confirmDelete && topic && (
        <LearningModal
          title={t("deleteTopic")}
          description={t("deleteTopicWarning")}
          onClose={() => setConfirmDelete(false)}
        >
          <div className="flex gap-3">
            <Button
              variant="secondary"
              className="flex-1"
              disabled={deleteTopic.isPending}
              onClick={() => setConfirmDelete(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              variant="destructive"
              className="flex-1"
              disabled={deleteTopic.isPending}
              onClick={() =>
                deleteTopic.mutate(topic.id, {
                  onSuccess: () => router.push("/learning"),
                })
              }
            >
              {t("delete")}
            </Button>
          </div>
          {deleteTopic.isError && (
            <p role="alert" className="text-destructive text-sm">
              {t("deleteError")}
            </p>
          )}
        </LearningModal>
      )}
    </div>
  )
}
