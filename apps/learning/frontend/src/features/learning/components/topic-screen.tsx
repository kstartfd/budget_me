"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconArrowLeft,
  IconCamera,
  IconChevronRight,
  IconKey,
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
import { EntryActionBar, EntryDetail } from "./entry-detail"
import { EntryForm } from "./entry-form"

type Kind = "all" | "word" | "phrase"
type Status = "all" | "inbox" | "learning" | "known"

const circleButton =
  "bg-secondary border-border text-muted-foreground size-[38px] rounded-full border"

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

  return (
    <div className="anim-fade space-y-[18px] lg:pt-3">
      {entryPage ? (
        <>
          <ScreenBar
            backLabel={topic.name}
            onBack={() => setSelectedId(null)}
            title={selected ? t(selected.kind) : t("entryDetails")}
            action={
              selected && (
                <Button
                  variant="secondary"
                  size="icon"
                  aria-label={t("editEntry")}
                  className={circleButton}
                  onClick={() => setEntryForm(selected)}
                >
                  <IconPencil size={16} />
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
                variant="secondary"
                size="icon"
                aria-label={t("editTopic")}
                className={circleButton}
                onClick={onEditTopic}
              >
                <IconSettings size={17} />
              </Button>
            }
          />
          <header className="hidden items-start justify-between gap-3 lg:flex">
            <div className="min-w-0">
              <Link
                href="/learning"
                className="text-muted-foreground inline-flex items-center gap-1 text-[13px]"
              >
                <IconArrowLeft size={14} />
                {t("title")}
              </Link>
              <h1 className="mt-1.5 text-[28px] font-bold tracking-[-0.02em] break-words">
                {topic.name}
              </h1>
              <p className="text-muted-foreground mt-0.5 text-[13px]">{pair}</p>
            </div>
            <Button
              variant="secondary"
              size="icon"
              aria-label={t("editTopic")}
              className={circleButton}
              onClick={onEditTopic}
            >
              <IconSettings size={18} />
            </Button>
          </header>

          <div className="flex min-w-0 flex-col gap-2.5 lg:flex-row lg:items-center lg:gap-3">
            <div
              className="flex flex-wrap gap-2 lg:gap-1 lg:rounded-xl lg:bg-card lg:p-1"
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
                    "h-auto rounded-full px-4 py-2 text-[13px] font-medium lg:rounded-[9px] lg:px-3.5 lg:py-[7px]",
                    status === value
                      ? "bg-primary/15 text-primary"
                      : "text-foreground/75",
                  )}
                >
                  {t(value)}
                </Button>
              ))}
            </div>
            <div
              className="bg-card flex min-w-0 gap-1 rounded-xl p-1"
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
                    "h-auto min-w-0 flex-1 rounded-[9px] px-2 py-[7px] text-[13px] font-medium lg:flex-none lg:px-3.5",
                    kind === value
                      ? "bg-secondary text-foreground"
                      : "text-muted-foreground",
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
            <div className="relative order-first min-w-0 lg:order-none lg:flex-1">
              <IconSearch
                size={16}
                className="text-faint absolute top-3 left-3.5"
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
              variant="secondary"
              onClick={() => setScanning(true)}
              className="text-primary hidden h-10 rounded-xl px-4 text-sm font-semibold lg:inline-flex"
            >
              <IconCamera size={17} />
              {scanT("open")}
            </Button>
            <Button
              onClick={() => setEntryForm("new")}
              className="text-primary-foreground hidden h-10 rounded-xl px-5 text-sm font-semibold lg:inline-flex"
            >
              <IconPlus size={17} />
              {t("newEntry")}
            </Button>
          </div>

          <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.05fr)]">
            <section className="min-w-0" aria-label={t("entries")}>
              {entries.isPending ? (
                <div className="bg-card border-border space-y-3 rounded-[18px] border p-5">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : entries.isError ? (
                <p
                  role="alert"
                  className="text-muted-foreground bg-card border-border rounded-[18px] border p-6 text-sm"
                >
                  {t("loadError")}
                </p>
              ) : rows.length === 0 ? (
                <div className="bg-card border-border rounded-[18px] border p-6">
                  <p className="text-sm font-semibold">
                    {t(filtered ? "noMatches" : "entriesEmpty")}
                  </p>
                  <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
                    {t(filtered ? "noMatchesHint" : "entriesEmptyHint")}
                  </p>
                  {!filtered && (
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
                <div className="divide-rowline flex flex-col gap-2 lg:gap-0 lg:divide-y lg:overflow-hidden lg:rounded-[18px] lg:border lg:border-border lg:bg-card lg:shadow-[var(--shadow-card)]">
                  {rows.map((entry) => (
                    <button
                      key={entry.id}
                      type="button"
                      onClick={() => setSelectedId(entry.id)}
                      className={cn(
                        "bg-card border-border flex w-full min-w-0 items-start gap-3 rounded-[14px] border px-3.5 py-3 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary lg:rounded-none lg:border-0 lg:px-[18px] lg:py-3.5 lg:hover:bg-secondary/60",
                        desktop &&
                          selectedId === entry.id &&
                          "lg:bg-primary/[0.07] lg:shadow-[inset_3px_0_0_var(--primary)] lg:hover:bg-primary/[0.07]",
                      )}
                    >
                      <span
                        className={cn(
                          "mt-1.5 size-2 shrink-0 rounded-full",
                          entry.status === "known"
                            ? "bg-success"
                            : entry.status === "learning"
                              ? "bg-primary"
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
                        <span className="text-faint mt-1.5 block truncate text-[11px]">
                          {[t(entry.kind), t(entry.status), entry.source]
                            .filter(Boolean)
                            .join(" · ")}
                        </span>
                      </div>
                      <IconChevronRight
                        className="text-faint mt-1 hidden shrink-0 lg:block"
                        size={16}
                      />
                    </button>
                  ))}
                </div>
              )}
              {entries.hasNextPage && (
                <div className="pt-3">
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
            <aside className="bg-card border-border sticky top-6 hidden max-h-[calc(100dvh-3rem)] min-h-80 min-w-0 flex-col overflow-y-auto rounded-[18px] border px-[26px] py-6 shadow-[var(--shadow-card)] lg:flex">
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
              className="text-muted-foreground inline-flex items-center gap-2 text-[13px]"
            >
              <IconKey size={15} />
              {t("tokens")}
            </Link>
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
                className="h-[50px] flex-1 rounded-[14px] text-base font-bold"
              >
                <IconPlus size={18} />
                {t("newEntry")}
              </Button>
              <Button
                variant="secondary"
                size="icon"
                aria-label={scanT("open")}
                onClick={() => setScanning(true)}
                className="text-primary size-[50px] shrink-0 rounded-[14px]"
              >
                <IconCamera size={22} />
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
