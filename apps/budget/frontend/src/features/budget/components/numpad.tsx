"use client"

/**
 * The sheets' 3×4 numpad. Value is the raw display string ("42,50") — comma
 * decimal, max two fraction digits, max 7 chars, leading zero replaced —
 * exactly the design's key logic. Parsing to minor units happens at save.
 */

const keyDefs = ["1", "2", "3", "4", "5", "6", "7", "8", "9", ",", "0", "del"]

export function pressKey(value: string, key: string): string {
  if (key === "del") return value.slice(0, -1)
  if (key === ",") {
    if (value.includes(",") || value === "") return value
    return value + ","
  }
  if (value.includes(",") && value.split(",")[1].length >= 2) return value
  if (value.length >= 7) return value
  const base = value === "0" ? "" : value
  return base + key
}

/** "42,50" → 4250; empty/garbage → 0. */
export function amountValueMinor(value: string): number {
  const parsed = Number.parseFloat((value || "0").replace(",", "."))
  return Number.isFinite(parsed) ? Math.round(parsed * 100) : 0
}

export function Numpad({
  value,
  onChange,
}: {
  value: string
  onChange: (next: string) => void
}) {
  return (
    <div className="grid grid-cols-3 gap-[9px]">
      {keyDefs.map((k) => (
        <button
          key={k}
          type="button"
          aria-label={k === "del" ? "⌫" : k}
          onClick={() => onChange(pressKey(value, k))}
          className="bg-secondary flex h-[52px] items-center justify-center rounded-[15px] text-[21px] font-semibold"
        >
          {k === "del" ? "⌫" : k}
        </button>
      ))}
    </div>
  )
}
