"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import {
  periodEndISO,
  periodOpenKey,
  periodStartISO,
  shiftMonth,
} from "@/lib/month"
import { cn } from "@/lib/utils"

import { useCategoryLabel } from "../category-label"
import type { Category } from "../schemas"
import { BottomSheet } from "./bottom-sheet"

export type TxDirectionFilter = "all" | "expense" | "income"

export type TxFilters = {
  from: string
  to: string
  direction: TxDirectionFilter
  category: string // "all" or a category id
}

/** Default range: the current billing period's start → today. */
export function defaultRange(
  today: string,
  startDay: number,
): { from: string; to: string } {
  return {
    from: periodStartISO(periodOpenKey(today, startDay), startDay),
    to: today,
  }
}

function addDaysISO(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)

  return d.toISOString().slice(0, 10)
}

/**
 * Bottom-sheet of all Transactions-screen filters: a date range (quick presets
 * + От/По pickers), a direction segment (all / expense / income) and a category
 * picker. Edits stage in a local draft; «Применить» commits it to the page and
 * closes, «Сбросить» returns the draft to the defaults.
 */
export function TransactionsFiltersSheet({
  categories,
  value,
  today,
  startDay,
  onApply,
  onClose,
}: {
  categories: Category[]
  value: TxFilters
  today: string
  startDay: number
  onApply: (next: TxFilters) => void
  onClose: () => void
}) {
  const t = useTranslations("txFilters")
  const label = useCategoryLabel()
  const [draft, setDraft] = useState<TxFilters>(value)

  const openKey = periodOpenKey(today, startDay)
  const prevKey = shiftMonth(openKey, -1)
  const presets = [
    {
      id: "period",
      label: t("presetPeriod"),
      from: periodStartISO(openKey, startDay),
      to: today,
    },
    {
      id: "prev",
      label: t("presetPrev"),
      from: periodStartISO(prevKey, startDay),
      to: periodEndISO(prevKey, startDay),
    },
    { id: "d30", label: t("preset30d"), from: addDaysISO(today, -29), to: today },
    { id: "all", label: t("presetAll"), from: "2000-01-01", to: today },
  ]
  const activePreset = presets.find(
    (p) => p.from === draft.from && p.to === draft.to,
  )?.id

  function setRange(from: string, to: string) {
    setDraft((d) => ({ ...d, from, to }))
  }

  const directions: TxDirectionFilter[] = ["all", "expense", "income"]

  return (
    <BottomSheet title={t("title")} onClose={onClose}>
      <FilterGroup label={t("range")}>
        <div className="flex flex-wrap gap-2">
          {presets.map((p) => (
            <Chip
              key={p.id}
              active={activePreset === p.id}
              onClick={() => setRange(p.from, p.to)}
            >
              {p.label}
            </Chip>
          ))}
        </div>
        <div className="mt-3 grid grid-cols-2 gap-2">
          <DateField
            label={t("from")}
            value={draft.from}
            max={draft.to}
            onChange={(v) => setRange(v, draft.to)}
          />
          <DateField
            label={t("to")}
            value={draft.to}
            min={draft.from}
            max={today}
            onChange={(v) => setRange(draft.from, v)}
          />
        </div>
      </FilterGroup>

      <FilterGroup label={t("direction")}>
        <div className="bg-secondary flex rounded-full p-1 font-medium">
          {directions.map((d) => (
            <button
              key={d}
              type="button"
              aria-pressed={draft.direction === d}
              onClick={() => setDraft((x) => ({ ...x, direction: d }))}
              className={cn(
                "flex-1 rounded-full py-2 text-[13px]",
                draft.direction === d
                  ? "bg-primary text-primary-foreground font-semibold"
                  : "text-muted-foreground",
              )}
            >
              {t(d)}
            </button>
          ))}
        </div>
      </FilterGroup>

      <FilterGroup label={t("category")}>
        <div className="scr chips -mx-0.5 flex gap-2 overflow-x-auto px-0.5 pb-1">
          <Chip
            active={draft.category === "all"}
            onClick={() => setDraft((x) => ({ ...x, category: "all" }))}
          >
            {t("allCategories")}
          </Chip>
          {categories.map((c) => (
            <Chip
              key={c.id}
              active={draft.category === c.id}
              color={c.color}
              onClick={() => setDraft((x) => ({ ...x, category: c.id }))}
            >
              {label(c)}
            </Chip>
          ))}
        </div>
      </FilterGroup>

      <div className="mt-5 flex gap-2.5">
        <button
          type="button"
          onClick={() =>
            setDraft({
              ...defaultRange(today, startDay),
              direction: "all",
              category: "all",
            })
          }
          className="border-border text-muted-foreground flex-1 rounded-2xl border p-3.5 text-[14.5px] font-semibold"
        >
          {t("reset")}
        </button>
        <button
          type="button"
          onClick={() => {
            onApply(draft)
            onClose()
          }}
          className="bg-primary text-primary-foreground flex-[1.4] rounded-2xl p-3.5 text-[14.5px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
        >
          {t("apply")}
        </button>
      </div>
    </BottomSheet>
  )
}

function FilterGroup({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="mt-4">
      <div className="text-faint mb-2 font-mono text-[10px] font-semibold tracking-[0.06em] uppercase">
        {label}
      </div>
      {children}
    </div>
  )
}

function Chip({
  active,
  color,
  onClick,
  children,
}: {
  active: boolean
  color?: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "flex shrink-0 items-center gap-1.5 rounded-full border px-3.5 py-[7px] text-[12.5px] font-medium whitespace-nowrap",
        active
          ? "bg-primary border-primary text-primary-foreground"
          : "bg-card border-border text-muted-foreground",
      )}
    >
      {color && (
        <span
          className="size-2 shrink-0 rounded-full"
          style={{ background: color }}
        />
      )}
      {children}
    </button>
  )
}

function DateField({
  label,
  value,
  min,
  max,
  onChange,
}: {
  label: string
  value: string
  min?: string
  max?: string
  onChange: (v: string) => void
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-faint text-[11px] font-medium">{label}</span>
      <input
        type="date"
        value={value}
        min={min}
        max={max}
        onChange={(e) => e.target.value && onChange(e.target.value)}
        className="border-border bg-secondary text-foreground rounded-xl border px-3 py-2.5 text-[13.5px] outline-none"
      />
    </label>
  )
}
