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
    <div className="anim-fade flex flex-col gap-4 lg:gap-6">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:gap-4">
        <Link
          href="/lists"
          className="text-primary flex items-center gap-0.5 self-start text-[15px] lg:hidden"
        >
          <ChevronLeftIcon size={16} />
          {t("title")}
        </Link>
        <Link
          href="/lists"
          aria-label={t("title")}
          className="border-border bg-card text-muted-foreground hidden size-10 shrink-0 items-center justify-center rounded-xl border lg:flex"
        >
          <ChevronLeftIcon />
        </Link>
        <div className="min-w-0">
          <div className="text-faint hidden truncate font-mono text-xs lg:block">
            {t("title")} › {label(list)}
          </div>
          <h1 className="flex items-center gap-2.5 text-[26px] font-bold tracking-[-0.02em] lg:mt-0.5">
            {list && (
              <span
                className="size-5 shrink-0 rounded-full"
                style={{
                  background: list.color,
                  boxShadow: "inset 0 0 0 1px rgba(255,255,255,0.18)",
                }}
              />
            )}
            <span className="truncate">{label(list)}</span>
          </h1>
        </div>
      </div>

      {items.data.length === 0 ? (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {t("emptyItems")}
        </p>
      ) : (
        <div className="flex flex-col gap-2.5 lg:gap-0 lg:overflow-hidden lg:rounded-[18px] lg:border lg:border-border lg:bg-card">
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
    <div aria-hidden className="flex flex-col gap-4 lg:gap-6">
      <div className="flex items-center gap-4">
        <Skeleton className="hidden size-10 shrink-0 rounded-xl lg:block" />
        <Skeleton className="h-8 w-40 rounded-lg" />
      </div>
      <div className="flex flex-col gap-2.5 lg:gap-0">
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-[84px] w-full rounded-2xl lg:h-[76px]" />
        ))}
      </div>
    </div>
  )
}
