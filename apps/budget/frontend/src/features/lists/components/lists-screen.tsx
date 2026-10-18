"use client"

import { useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { PencilIcon, PlusIcon } from "@/components/app-icons"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

import { useLists } from "../queries"
import { useListLabel } from "../list-label"
import type { ItemList } from "../schemas"
import { ListSheet } from "./list-sheet"

/**
 * `/lists` — the list of lists (design «Списки» 2a desktop / 2c Telegram):
 * preset cards (Аптечка, Косметика) plus any personal ones, each a link into
 * /lists/[id] with its item count in the corner (shown once the backend sends
 * `itemCount`). Creating a list has its own explicit control rather than
 * claiming the nav's ＋ (which belongs to adding an ITEM once a list is
 * open): a button under the grid on desktop, a dashed card in the grid on a
 * phone. Only a personal list (slug === null) gets an edit pencil — a preset
 * has no `name` to rename and the backend 404s on it anyway (WHERE user_id =
 * self can never match a preset row).
 */
export function ListsScreen() {
  const t = useTranslations("lists")
  const label = useListLabel()
  const lists = useLists()
  const [sheet, setSheet] = useState<ItemList | "new" | null>(null)

  return (
    <div className="anim-fade flex flex-col gap-[18px] lg:gap-6">
      <div>
        <h1 className="text-[26px] font-bold tracking-[-0.02em] lg:text-[28px]">
          {t("title")}
        </h1>
        <p className="text-muted-foreground mt-1.5 text-[13px] leading-[1.4] lg:mt-1 lg:text-sm">
          {t("desc")}
        </p>
      </div>

      {lists.isPending ? (
        <GridSkeleton />
      ) : lists.isError ? (
        <p role="alert" className="text-muted-foreground py-10 text-center text-sm">
          {t("loadError")}
        </p>
      ) : (
        <div className="grid grid-cols-2 content-start gap-2.5 lg:grid-cols-3 lg:gap-3.5">
          {lists.data.map((l) => (
            <div key={l.id} className="relative">
              <Link
                href={`/lists/${l.id}`}
                className="bg-card border-border flex h-full min-h-[96px] flex-col gap-[18px] rounded-[18px] border p-3.5 shadow-[var(--shadow-card)] transition-colors hover:border-[color:var(--primary-line)] lg:gap-[22px] lg:p-[18px]"
              >
                <span className="flex items-center justify-between">
                  <span
                    className="size-8 rounded-full lg:size-9"
                    style={{
                      background: l.color,
                      boxShadow:
                        "inset 0 0 0 1px rgba(255,255,255,0.18), 0 1px 3px rgba(0,0,0,0.18)",
                    }}
                  />
                  {l.itemCount !== undefined && (
                    <span className="text-faint font-mono text-xs">
                      {l.itemCount}
                    </span>
                  )}
                </span>
                <span
                  className={cn(
                    "truncate text-[15px] font-semibold",
                    l.slug === null && "pr-8",
                  )}
                >
                  {label(l)}
                </span>
              </Link>

              {l.slug === null && (
                <button
                  type="button"
                  aria-label={t("editListAria", { name: label(l) })}
                  onClick={() => setSheet(l)}
                  className="border-border bg-secondary text-muted-foreground hover:text-foreground absolute right-2.5 bottom-2.5 flex size-7 items-center justify-center rounded-full border transition-colors"
                >
                  <PencilIcon />
                </button>
              )}
            </div>
          ))}

          <button
            type="button"
            onClick={() => setSheet("new")}
            className="text-primary flex min-h-[90px] items-center justify-center gap-1.5 rounded-[18px] border border-dashed border-[color:var(--primary-line)] p-3.5 text-sm font-semibold lg:hidden"
          >
            <PlusIcon size={14} strokeWidth={2.4} />
            {t("newListShort")}
          </button>
        </div>
      )}

      <button
        type="button"
        onClick={() => setSheet("new")}
        className="bg-primary text-primary-foreground hidden items-center gap-2 self-start rounded-[14px] px-6 py-[13px] text-sm font-semibold lg:flex"
      >
        <PlusIcon size={14} strokeWidth={2.4} />
        {t("newList")}
      </button>

      {sheet && (
        <ListSheet
          list={sheet === "new" ? undefined : sheet}
          onClose={() => setSheet(null)}
        />
      )}
    </div>
  )
}

function GridSkeleton() {
  return (
    <div aria-hidden className="grid grid-cols-2 gap-2.5 lg:grid-cols-3 lg:gap-3.5">
      {Array.from({ length: 3 }, (_, i) => (
        <div
          key={i}
          className="bg-card border-border flex min-h-[96px] flex-col justify-between gap-[18px] rounded-[18px] border p-3.5 lg:p-[18px]"
        >
          <Skeleton className="size-8 rounded-full lg:size-9" />
          <Skeleton className="h-4 w-20 rounded" />
        </div>
      ))}
    </div>
  )
}
