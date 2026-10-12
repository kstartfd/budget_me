"use client"

import { useEffect, useRef, useSyncExternalStore } from "react"

/**
 * Drum / wheel day picker (the design's "2a wheel"): a vertical scroll-snap
 * strip of days, the centered row highlighted and enlarged, neighbours fading
 * out with distance. Driven by native scrolling so touch, trackpad and mouse
 * wheel all work for free; also focusable with full keyboard control.
 *
 * Controlled: renders `value`, reports the settled day through `onChange`.
 * Values are 1–28 (see settings schema); days 29–31 are intentionally absent.
 * Honours prefers-reduced-motion by dropping snap animation and smooth scroll.
 */

const ROW_H = 38
const VISIBLE = 5
/** Rows above/below centre — also the strip's top/bottom padding. */
const PAD = ((VISIBLE - 1) / 2) * ROW_H
const HEIGHT = VISIBLE * ROW_H
const MASK =
  "linear-gradient(180deg, transparent, #000 22%, #000 78%, transparent)"

const MIN_DAY = 1
const MAX_DAY = 28
const DAYS = Array.from({ length: MAX_DAY - MIN_DAY + 1 }, (_, i) => i + MIN_DAY)

function clampDay(day: number): number {
  return Math.min(Math.max(day, MIN_DAY), MAX_DAY)
}

/** `true` when the OS asks for reduced motion. Hydration-safe (server: false). */
function usePrefersReducedMotion(): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mq = window.matchMedia("(prefers-reduced-motion: reduce)")
      mq.addEventListener("change", onChange)
      return () => mq.removeEventListener("change", onChange)
    },
    () => window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    () => false,
  )
}

export function DayWheel({
  value,
  onChange,
  label,
}: {
  value: number
  onChange: (day: number) => void
  /** Accessible name for the slider. */
  label: string
}) {
  const listRef = useRef<HTMLDivElement>(null)
  const settleTimer = useRef<number | null>(null)
  const didInit = useRef(false)
  const reduced = usePrefersReducedMotion()

  // Keep the scroll position in sync with the controlled value (initial mount,
  // clicks and keyboard jumps). No-op when the user just scrolled here, so the
  // scroll handler below can't fight this effect.
  useEffect(() => {
    const el = listRef.current
    if (!el) return
    const target = (value - MIN_DAY) * ROW_H
    const behavior = didInit.current && !reduced ? "smooth" : "auto"
    didInit.current = true
    if (Math.abs(el.scrollTop - target) > 1) {
      el.scrollTo({ top: target, behavior })
    }
  }, [value, reduced])

  useEffect(() => {
    return () => {
      if (settleTimer.current) window.clearTimeout(settleTimer.current)
    }
  }, [])

  function handleScroll() {
    if (settleTimer.current) window.clearTimeout(settleTimer.current)
    settleTimer.current = window.setTimeout(() => {
      const el = listRef.current
      if (!el) return
      const day = clampDay(Math.round(el.scrollTop / ROW_H) + MIN_DAY)
      if (day !== value) onChange(day)
    }, 90)
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    let next = value
    switch (event.key) {
      case "ArrowUp":
        next = value - 1
        break
      case "ArrowDown":
        next = value + 1
        break
      case "PageUp":
        next = value - 3
        break
      case "PageDown":
        next = value + 3
        break
      case "Home":
        next = MIN_DAY
        break
      case "End":
        next = MAX_DAY
        break
      default:
        return
    }
    event.preventDefault()
    next = clampDay(next)
    if (next !== value) onChange(next)
  }

  return (
    <div
      role="slider"
      tabIndex={0}
      aria-label={label}
      aria-orientation="vertical"
      aria-valuemin={MIN_DAY}
      aria-valuemax={MAX_DAY}
      aria-valuenow={value}
      aria-valuetext={String(value)}
      onKeyDown={handleKeyDown}
      className="relative rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--ring)]"
      style={{ height: HEIGHT }}
    >
      {/* Centre highlight band the numbers scroll under. */}
      <div
        aria-hidden
        className="bg-[color:var(--primary-wash)] pointer-events-none absolute inset-x-0 rounded-xl"
        style={{ top: PAD, height: ROW_H }}
      />
      <div
        ref={listRef}
        aria-hidden
        onScroll={handleScroll}
        className="scr relative h-full overflow-y-auto"
        style={{
          scrollSnapType: reduced ? undefined : "y mandatory",
          WebkitMaskImage: MASK,
          maskImage: MASK,
        }}
      >
        <div style={{ paddingTop: PAD, paddingBottom: PAD }}>
          {DAYS.map((n) => {
            const selected = n === value
            const dist = Math.abs(n - value)
            return (
              <button
                key={n}
                type="button"
                tabIndex={-1}
                onClick={() => {
                  if (n !== value) onChange(n)
                }}
                className="flex w-full items-center justify-center border-none bg-transparent p-0"
                style={{
                  height: ROW_H,
                  scrollSnapAlign: "center",
                  fontWeight: selected ? 700 : 600,
                  fontSize: selected ? 22 : 16,
                  color: selected ? "var(--primary)" : "var(--foreground)",
                  opacity: selected ? 1 : Math.max(0.2, 1 - dist * 0.28),
                  transition: reduced
                    ? undefined
                    : "font-size 0.16s ease, opacity 0.16s ease",
                }}
              >
                {n}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
