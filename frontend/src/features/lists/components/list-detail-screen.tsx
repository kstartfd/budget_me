"use client"

import { useCallback, useMemo, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { useProvideAddAction } from "@/components/add-action"
import { ChevronLeftIcon } from "@/components/app-icons"
import { Skeleton } from "@/components/ui/skeleton"

import { useListItems, useLists } from "../queries"
import { useListLabel } from "../list-label"
import { ItemRow } from "./item-row"
import { ItemDetailSheet } from "./item-detail-sheet"
import { ItemSheet } from "./item-sheet"
import type { Item } from "../schemas"

/**
 * `/lists/[id]` — one list's items (design PHONE «Аптечка» / WEB «список
 * таблицей»): a breadcrumb back to Списки, the coloured title, then rows.
 * Unlike the top-level screen, THIS one claims the nav's ＋ — adding an item
 * only makes sense once a specific list is open, same reasoning as День
 * owning ＋ for a new task.
 */
export function ListDetailScreen({ listId }: { listId: string }) {
  const t = useTranslations("lists")
  const label = useListLabel()

  const lists = useLists()
  const items = useListItems(listId)
  const [editing, setEditing] = useState<Item | "add" | null>(null)
  const [selected, setSelected] = useState<Item | null>(null)

  useProvideAddAction(useCallback(() => setEditing("add"), []))

  const list = useMemo(
    () => lists.data?.find((l) => l.id === listId),
    [lists.data, listId],
  )

  if (items.isPending || lists.isPending) return <DetailSkeleton />

  if (items.isError || lists.isError) {
    return (
      <p
        role="alert"
        className="text-muted-foreground py-10 text-center text-sm"
      >
        {t("loadError")}
      </p>
    )
  }

  return (
    <div className="anim-fade flex flex-col gap-[18px] lg:max-w-3xl">
      <div className="flex items-start gap-2.5">
        <Link
          href="/lists"
          aria-label={t("title")}
          className="border-border bg-secondary text-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px] border"
        >
          <ChevronLeftIcon />
        </Link>
        <div className="min-w-0">
          <div className="text-faint truncate font-mono text-[11px]">
            {t("title")} › {label(list)}
          </div>
          <div className="flex items-center gap-2.5">
            {list && (
              <span
                className="size-[22px] shrink-0 rounded-full"
                style={{
                  background: list.color,
                  boxShadow: "inset 0 0 0 1px rgba(255,255,255,0.18)",
                }}
              />
            )}
            <h1 className="truncate text-[22px] font-bold tracking-[-0.02em]">
              {label(list)}
            </h1>
          </div>
        </div>
      </div>

      {items.data.length === 0 ? (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {t("emptyItems")}
        </p>
      ) : (
        <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[22px] border shadow-[var(--shadow-card)]">
          {items.data.map((item) => (
            <ItemRow
              key={item.id}
              item={item}
              onOpen={() => setSelected(item)}
            />
          ))}
        </div>
      )}

      {selected && (
        <ItemDetailSheet
          item={selected}
          listId={listId}
          onClose={() => setSelected(null)}
          onEdit={() => {
            setSelected(null)
            setEditing(selected)
          }}
        />
      )}

      {editing && (
        <ItemSheet
          listId={listId}
          item={editing === "add" ? undefined : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  )
}

function DetailSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-[18px] lg:max-w-3xl">
      <div className="flex items-center gap-2.5">
        <Skeleton className="size-[34px] shrink-0 rounded-[11px]" />
        <Skeleton className="h-6 w-32 rounded" />
      </div>
      <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[22px] border">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="flex items-center gap-3 px-[15px] py-[13px]">
            <div className="min-w-0 flex-1">
              <Skeleton className="h-4 w-32 rounded" />
              <Skeleton className="mt-1.5 h-3 w-20 rounded" />
            </div>
            <Skeleton className="h-4 w-20 rounded" />
          </div>
        ))}
      </div>
    </div>
  )
}
