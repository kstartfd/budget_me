"use client"

import { useRef } from "react"

import { cn } from "@/lib/utils"

/**
 * Calendar-style day picker (the design's "2b calendar"): a 7-wide grid of day
 * numbers 1–28, click to select, the chosen day accented. The desktop
 * counterpart to DayWheel — the wheel reads as an iOS drum that's unclear on a
 * wide pointer screen, so lg+ shows this grid instead (settings picks one by
 * breakpoint). Same controlled `value` / `onChange` contract as the wheel.
 *
 * Values are 1–28 (see settings schema); days 29–31 clamp to the month's
 * length (shown as a hint by the settings screen). A roving-tabindex
 * radiogroup: one tab stop, arrow keys move (and select) across the grid.
 */

const MIN_DAY = 1
const MAX_DAY = 28
const COLS = 7
const DAYS = Array.from({ length: MAX_DAY - MIN_DAY + 1 }, (_, i) => i + MIN_DAY)

function clampDay(day: number): number {
  return Math.min(Math.max(day, MIN_DAY), MAX_DAY)
}

export function DayGrid({
  value,
  onChange,
  label,
}: {
  value: number
  onChange: (day: number) => void
  /** Accessible name for the radiogroup. */
  label: string
}) {
  const gridRef = useRef<HTMLDivElement>(null)

  function focusDay(day: number) {
    gridRef.current
      ?.querySelector<HTMLButtonElement>(`[data-day="${day}"]`)
      ?.focus()
  }

  function move(next: number) {
    const day = clampDay(next)
    if (day !== value) onChange(day)
    focusDay(day)
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    switch (event.key) {
      case "ArrowLeft":
        move(value - 1)
        break
      case "ArrowRight":
        move(value + 1)
        break
      case "ArrowUp":
        move(value - COLS)
        break
      case "ArrowDown":
        move(value + COLS)
        break
      case "Home":
        move(MIN_DAY)
        break
      case "End":
        move(MAX_DAY)
        break
      default:
        return
    }
    event.preventDefault()
  }

  return (
    <div
      ref={gridRef}
      role="radiogroup"
      aria-label={label}
      onKeyDown={handleKeyDown}
      className="grid grid-cols-7 gap-1.5"
    >
      {DAYS.map((n) => {
        const selected = n === value
        return (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={selected}
            data-day={n}
            tabIndex={selected ? 0 : -1}
            onClick={() => {
              if (n !== value) onChange(n)
            }}
            className={cn(
              "flex h-9 items-center justify-center rounded-lg text-[13px] font-semibold outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--ring)]",
              selected
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-secondary",
            )}
          >
            {n}
          </button>
        )
      })}
    </div>
  )
}
