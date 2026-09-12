"use client"

import { useState } from "react"

/**
 * 5 stars with half-steps standing in for a 1..10 rating (rating = stars×2) —
 * 10 stars in a row don't read on a 390px screen, 5 with halves keep the
 * familiar shape and the full range. Two flat SVG layers (empty behind,
 * filled in front, clipped to a % width) rather than per-star fill math — the
 * filled layer's width alone draws every half-star position.
 *
 * Interactive when `onChange` is given: 10 invisible half-star hit targets
 * overlay the same row, so a tap always lands on a whole half-step.
 */
const starPath =
  "M12 2.5l2.9 6.06 6.6.7-4.9 4.55 1.28 6.6L12 16.9l-5.88 3.5 1.28-6.6-4.9-4.55 6.6-.7z"

function StarGlyph({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
      <path d={starPath} />
    </svg>
  )
}

export function StarRating({
  rating,
  size = 17,
  onChange,
}: {
  /** 1..10, or null when not rated yet — renders as 0 filled stars. */
  rating: number | null
  size?: number
  /** Present → interactive (the add/edit sheet); absent → read-only display. */
  onChange?: (rating: number) => void
}) {
  const [hover, setHover] = useState<number | null>(null)
  const value = hover ?? rating ?? 0
  const pct = (value / 10) * 100

  return (
    <span
      className="relative inline-flex shrink-0"
      style={{ width: size * 5, height: size }}
      onMouseLeave={() => setHover(null)}
    >
      <span className="text-faint absolute inset-0 flex">
        {Array.from({ length: 5 }, (_, i) => (
          <StarGlyph key={i} size={size} />
        ))}
      </span>
      <span
        aria-hidden
        className="text-warn absolute inset-0 flex overflow-hidden"
        style={{ width: `${pct}%` }}
      >
        {Array.from({ length: 5 }, (_, i) => (
          <StarGlyph key={i} size={size} />
        ))}
      </span>
      {onChange && (
        <span className="absolute inset-0 flex">
          {Array.from({ length: 10 }, (_, i) => {
            const half = i + 1

            return (
              <button
                key={half}
                type="button"
                aria-label={`${half}/10`}
                onMouseEnter={() => setHover(half)}
                onClick={() => onChange(half)}
                style={{ width: size / 2 }}
              />
            )
          })}
        </span>
      )}
    </span>
  )
}
