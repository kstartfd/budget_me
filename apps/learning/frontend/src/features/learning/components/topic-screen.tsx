"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconArrowLeft,
  IconCamera,
  IconChevronRight,
  IconLanguage,
  IconPencil,
  IconPlus,
  IconSearch,
  IconSettings,
  IconTrash,
} from "@tabler/icons-react"
import { BottomBar } from "@/components/bottom-bar"
import { useClaimFullScreen } from "@/components/full-screen"
import { ScreenBar } from "@/components/screen-bar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ScanModal } from "@/features/scan"
import { useIsDesktop } from "@/hooks/use-is-desktop"
import { cn } from "@/lib/utils"
import { useEntries, useEntry } from "../hooks"
import { useLanguageName } from "../language"
import type { Entry, Topic } from "../schemas"
import { EntryAudioButton } from "./audio-button"
import { EntryActionBar, EntryDetail } from "./entry-detail"
import { EntryForm } from "./entry-form"

type Kind = "all" | "word" | "phrase"
type Status = "all" | "inbox" | "learning" | "known"

const circleButton =
  "border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground size-9 shrink-0 rounded-md border"

const segmented = "bg-muted flex h-9 min-w-0 items-center gap-0.5 rounded-md p-1"
const segment = (active: boolean) =>
  cn(
    "h-full min-w-0 flex-1 rounded-sm px-3 text-[13px] font-medium transition-colors lg:flex-none lg:px-3.5",
    active
      ? "bg-background text-foreground"
      : "text-muted-foreground hover:text-foreground",
  )

export function TopicScreen({
  topic,
  onEditTopic,
  onDeleteTopic,
}: {
  topic: Topic
  onEditTopic: () => void
  onDeleteTopic: () => void
}) {
  const t = useTranslations("learning")
  const scanT = useTranslations("scan")
  const desktop = useIsDesktop()
  const languageName = useLanguageName()
  const [entryForm, setEntryForm] = useState<Entry | "new" | null>(null)
  const [scanning, setScanning] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [kind, setKind] = useState<Kind>("all")
  const [status, setStatus] = useState<Status>("all")
  const entries = useEntries(topic.id, { kind, status, search })
  const rows = entries.data?.pages.flat() ?? []
  const selectedEntry = useEntry(
    selectedId,
    rows.find((entry) => entry.id === selectedId),
  )
  const selected = selectedEntry.isError ? undefined : selectedEntry.data
  const pair = `${languageName(topic.language)} → ${languageName(topic.translationLanguage)}`
  const filtered = search !== "" || kind !== "all" || status !== "all"
  const entryPage = !desktop && selectedId !== null
  useClaimFullScreen()

  const { hasNextPage, isFetchingNextPage, isError, fetchNextPage } = entries
  const sentinelRef = useRef<HTMLDivElement>(null)
  const canLoadMore = hasNextPage && !isFetchingNextPage && !isError
  useEffect(() => {
    const node = sentinelRef.current
    if (!node || !canLoadMore || typeof IntersectionObserver === "undefined") {
      return
    }
    const observer = new IntersectionObserver(
      (items) => {
        if (items.some((item) => item.isIntersecting)) void fetchNextPage()
      },
      { rootMargin: "300px" },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [canLoadMore, fetchNextPage, rows.length])

  return (
    <div className="anim-fade space-y-4 lg:space-y-5">
      {entryPage ? (
        <>
          <ScreenBar
            backLabel={topic.name}
            onBack={() => setSelectedId(null)}
            title={selected ? t(selected.kind) : t("entryDetails")}
            action={
              selected && (
                <Button
                  variant="outline"
                  size="icon-lg"
                  aria-label={t("editEntry")}
                  className={circleButton}
                  onClick={() => setEntryForm(selected)}
                >
                  <IconPencil size={15} />
                </Button>
              )
            }
          />
          {selected ? (
            <>
              <EntryDetail
                key={selected.id}
                entry={selected}
                variant="page"
                onEdit={() => setEntryForm(selected)}
                onDelete={() => setSelectedId(null)}
              />
              <EntryActionBar
                entry={selected}
                onDelete={() => setSelectedId(null)}
              />
            </>
          ) : selectedEntry.isError ? (
            <p role="alert" className="text-muted-foreground text-sm">
              {t("loadError")}
            </p>
          ) : (
            <Skeleton className="h-40 w-full" />
          )}
        </>
      ) : (
        <>
          <ScreenBar
            className="lg:hidden"
            backLabel={t("title")}
            backHref="/learning"
            title={topic.name}
            action={
              <Button
                variant="outline"
                size="icon-lg"
                aria-label={t("editTopic")}
                className={circleButton}
                onClick={onEditTopic}
              >
                <IconSettings size={15} />
              </Button>
            }
          />
          <header className="hidden items-start justify-between gap-3 lg:flex">
            <div className="min-w-0">
              <Link
                href="/learning"
                className="text-muted-foreground hover:text-foreground inline-flex items-center gap-2 text-[13px] font-medium"
              >
                <IconArrowLeft size={14} />
                {t("title")}
              </Link>
              <h1 className="mt-4 text-3xl font-semibold tracking-[-0.01em] break-words">
                {topic.name}
              </h1>
              <p className="text-muted-foreground mt-1 text-[13px]">{pair}</p>
            </div>
            <Button
              variant="outline"
              size="icon-lg"
              aria-label={t("editTopic")}
              className={circleButton}
              onClick={onEditTopic}
            >
              <IconSettings size={15} />
            </Button>
          </header>

          <div className="flex min-w-0 flex-col gap-2.5 lg:flex-row lg:items-center lg:gap-3">
            <div className={segmented} role="group" aria-label={t("status")}>
              {(["all", "inbox", "learning", "known"] as const).map((value) => (
                <button
                  key={value}
                  type="button"
                  aria-pressed={status === value}
                  onClick={() => setStatus(value)}
                  className={segment(status === value)}
                >
                  {t(value)}
                </button>
              ))}
            </div>
            <div className={segmented} role="group" aria-label={t("kind")}>
              {(["all", "word", "phrase"] as const).map((value) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setKind(value)}
                  aria-pressed={kind === value}
                  className={segment(kind === value)}
                >
                  {t(
                    value === "all"
                      ? "allKinds"
                      : value === "word"
                        ? "words"
                        : "phrases",
                  )}
                </button>
              ))}
            </div>
            <div className="relative order-first min-w-0 lg:order-none lg:flex-1">
              <IconSearch
                size={15}
                className="text-faint pointer-events-none absolute top-1/2 left-3 -translate-y-1/2"
              />
              <Input
                aria-label={t("search")}
                placeholder={t("search")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="bg-background h-9 pl-9 text-sm md:text-sm"
              />
            </div>
            <Button
              variant="outline"
              size="lg"
              onClick={() => setScanning(true)}
              className="hidden px-3.5 text-[13px] lg:inline-flex"
            >
              <IconCamera size={15} />
              {scanT("open")}
            </Button>
            <Button
              size="lg"
              onClick={() => setEntryForm("new")}
              className="hidden px-3.5 text-[13px] lg:inline-flex"
            >
              <IconPlus size={14} />
              {t("newEntry")}
            </Button>
          </div>

          <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.05fr)]">
            <section className="min-w-0" aria-label={t("entries")}>
              {entries.isPending ? (
                <div className="bg-card border-border space-y-3 rounded-lg border p-5">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : entries.isError ? (
                <p
                  role="alert"
                  className="text-muted-foreground bg-card border-border rounded-lg border p-6 text-sm"
                >
                  {t("loadError")}
                </p>
              ) : rows.length === 0 ? (
                <div className="bg-card border-border rounded-lg border p-6">
                  <p className="text-sm font-semibold">
                    {t(filtered ? "noMatches" : "entriesEmpty")}
                  </p>
                  <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
                    {t(filtered ? "noMatchesHint" : "entriesEmptyHint")}
                  </p>
                  {!filtered && (
                    <Button
                      variant="outline"
                      size="lg"
                      className="mt-4 px-3.5 text-[13px]"
                      onClick={() => setEntryForm("new")}
                    >
                      <IconPlus size={14} />
                      {t("newEntry")}
                    </Button>
                  )}
                </div>
              ) : (
                <div className="divide-border border-border bg-background flex flex-col divide-y overflow-hidden rounded-lg border">
                  {rows.map((entry) => (
                    <div
                      key={entry.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => setSelectedId(entry.id)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault()
                          setSelectedId(entry.id)
                        }
                      }}
                      className={cn(
                        "hover:bg-muted/50 focus-visible:ring-ring flex w-full min-w-0 cursor-pointer items-start gap-3 px-4 py-3.5 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset",
                        desktop && selectedId === entry.id && "bg-muted hover:bg-muted",
                      )}
                    >
                      <span
                        className={cn(
                          "mt-1.5 size-2 shrink-0 rounded-full",
                          entry.status === "known"
                            ? "bg-success"
                            : entry.status === "learning"
                              ? "bg-info"
                              : "bg-warn",
                        )}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="line-clamp-2 text-sm leading-snug font-semibold break-words">
                          {entry.text}
                        </div>
                        <p className="text-muted-foreground mt-1 line-clamp-2 text-xs leading-snug break-words lg:text-[13px]">
                          {entry.translation || t("unfilled")}
                        </p>
                        <span className="text-faint mt-1 block truncate text-xs">
                          {[t(entry.kind), t(entry.status), entry.source]
                            .filter(Boolean)
                            .join(" · ")}
                        </span>
                      </div>
                      <EntryAudioButton entryId={entry.id} compact />
                      <IconChevronRight
                        className="text-faint mt-1 hidden shrink-0 lg:block"
                        size={16}
                      />
                    </div>
                  ))}
                </div>
              )}
              {entries.hasNextPage && (
                <div className="pt-3" ref={sentinelRef}>
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
            <aside className="bg-background border-border sticky top-6 hidden max-h-[calc(100dvh-3rem)] min-h-80 min-w-0 flex-col overflow-y-auto rounded-lg border px-6 py-6 lg:flex">
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

          <div className="flex items-center justify-end gap-3 pt-1">
            <Button
              variant="ghost"
              className="text-muted-foreground text-[13px]"
              onClick={onDeleteTopic}
            >
              <IconTrash size={14} />
              {t("deleteTopic")}
            </Button>
          </div>

          <BottomBar>
            <div className="flex gap-2">
              <Button
                onClick={() => setEntryForm("new")}
                className="h-11 flex-1 text-sm"
              >
                <IconPlus size={16} />
                {t("newEntry")}
              </Button>
              <Button
                variant="outline"
                size="icon"
                aria-label={scanT("open")}
                onClick={() => setScanning(true)}
                className="size-11 shrink-0"
              >
                <IconCamera size={18} />
              </Button>
            </div>
          </BottomBar>
        </>
      )}

      {scanning && (
        <ScanModal topicId={topic.id} onClose={() => setScanning(false)} />
      )}

      {entryForm && (
        <EntryForm
          entry={entryForm === "new" ? undefined : entryForm}
          topicId={topic.id}
          kind={kind === "word" ? "word" : "phrase"}
          onClose={() => setEntryForm(null)}
        />
      )}
    </div>
  )
}
