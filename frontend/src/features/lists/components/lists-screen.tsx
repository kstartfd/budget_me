"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { PlusIcon } from "@/components/app-icons"
import { Skeleton } from "@/components/ui/skeleton"

import { useLists } from "../queries"
import { useListLabel } from "../list-label"
import { ListSheet } from "./list-sheet"

/**
 * `/lists` — the list of lists (design PHONE «Списки»): preset cards
 * (Аптечка, Косметика) plus any personal ones, each a link into
 * /lists/[id]. Creating a list has its own explicit button here rather than
 * claiming the nav's ＋ (which belongs to adding an ITEM once a list is
 * open) — same choice as CategoriesScreen's own inline add button.
 */
export function ListsScreen() {
  const t = useTranslations("lists")
  const label = useListLabel()
  const lists = useLists()
  const [creating, setCreating] = useState(false)

  return (
    <div className="anim-fade flex flex-col gap-[18px]">
      <div>
        <h1 className="text-[22px] font-bold tracking-[-0.02em]">{t("title")}</h1>
        <p className="text-muted-foreground mt-1 text-[13px]">{t("desc")}</p>
      </div>

      {lists.isPending ? (
        <GridSkeleton />
      ) : lists.isError ? (
        <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
          {t("loadError")}
        </p>
      ) : (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {lists.data.map((l) => (
            <Link
              key={l.id}
              href={`/lists/${l.id}`}
              className="bg-card border-border flex flex-col gap-3 rounded-[22px] border p-4 shadow-[var(--shadow-card)]"
            >
              <span
                className="size-9 rounded-full"
                style={{
                  background: l.color,
                  boxShadow:
                    "inset 0 0 0 1px rgba(255,255,255,0.18), 0 1px 3px rgba(0,0,0,0.18)",
                }}
              />
              <span className="truncate text-[14.5px] font-semibold">
                {label(l)}
              </span>
            </Link>
          ))}
        </div>
      )}

      <button
        type="button"
        onClick={() => setCreating(true)}
        className="bg-primary text-primary-foreground flex w-full items-center justify-center gap-2 rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] lg:max-w-[260px]"
      >
        <PlusIcon />
        {t("newList")}
      </button>

      {creating && <ListSheet onClose={() => setCreating(false)} />}
    </div>
  )
}

function GridSkeleton() {
  return (
    <div aria-hidden className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {Array.from({ length: 4 }, (_, i) => (
        <div
          key={i}
          className="bg-card border-border flex flex-col gap-3 rounded-[22px] border p-4"
        >
          <Skeleton className="size-9 rounded-full" />
          <Skeleton className="h-4 w-16 rounded" />
        </div>
      ))}
    </div>
  )
}
