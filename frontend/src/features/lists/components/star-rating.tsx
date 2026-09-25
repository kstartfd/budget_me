"use client"

import { useState } from "react"

/** Five stars represent a 1..10 rating in half-star steps. */
const starPath =
  "M12 2.5l2.9 6.06 6.6.7-4.9 4.55 1.28 6.6L12 16.9l-5.88 3.5 1.28-6.6-4.9-4.55 6.6-.7z"

function StarGlyph({ size }: { size: number }) {
  return (
    <svg
      className="block"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
    >
      <path d={starPath} />
    </svg>
  )
}

export function StarRating({
  rating,
  size = 17,
  onChange,
}: {
  rating: number | null
  size?: number
  onChange?: (rating: number) => void
}) {
  const [hover, setHover] = useState<number | null>(null)
  const value = hover ?? rating ?? 0

  return (
    <span
      className="inline-flex shrink-0"
      style={{ width: size * 5, height: size }}
      onPointerLeave={() => setHover(null)}
    >
      {Array.from({ length: 5 }, (_, index) => {
        // Clip each star on its own so its fill stays aligned at any rating.
        const fill = Math.max(0, Math.min(2, value - index * 2)) * 50

        return (
          <span
            key={index}
            className="relative block shrink-0 text-faint"
            style={{ width: size, height: size }}
          >
            <StarGlyph size={size} />
            <span
              aria-hidden
              className="text-warn absolute inset-y-0 left-0 overflow-hidden"
              style={{ width: `${fill}%` }}
            >
              <span className="block" style={{ width: size }}>
                <StarGlyph size={size} />
              </span>
            </span>
            {onChange && (
              <span className="absolute inset-0 flex">
                {[0, 1].map((half) => {
                  const score = index * 2 + half + 1

                  return (
                    <button
                      key={score}
                      type="button"
                      className="h-full w-1/2 focus-visible:relative focus-visible:outline-2 focus-visible:outline-primary"
                      aria-label={`${score}/10`}
                      aria-pressed={rating === score}
                      onPointerEnter={(event) => {
                        if (event.pointerType === "mouse") setHover(score)
                      }}
                      onClick={() => onChange(score)}
                    />
                  )
                })}
              </span>
            )}
          </span>
        )
      })}
    </span>
  )
}
