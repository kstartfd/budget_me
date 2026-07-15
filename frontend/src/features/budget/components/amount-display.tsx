"use client"

import type { Direction } from "../schemas"
import { amountValueMinor } from "./numpad"
import { useBudgetUI } from "./ui-context"

/**
 * The sheets' centered amount readout: big bold number, "Br" unit, and the
 * converted companion line (absent when the currency is BYN). Grey while
 * empty, ink once typing starts. When a `direction` is given (the add sheet)
 * it prefixes the sign and tints income green; limit-sheet omits it and
 * renders the plain number as before.
 */
export function AmountDisplay({
  value,
  size = 44,
  direction,
}: {
  value: string
  size?: number
  /** When set, show the +/− sign and tint income green. */
  direction?: Direction
}) {
  const { fx } = useBudgetUI()
  const approx = fx(amountValueMinor(value))

  const empty = value === ""
  const sign = direction === "income" ? "+" : direction === "expense" ? "−" : ""
  const color = empty
    ? "var(--faint)"
    : direction === "income"
      ? "var(--success)"
      : "var(--foreground)"
  return (
    <div className="text-center">
      <div className="inline-flex items-baseline gap-2">
        <span
          className="font-bold tracking-[-0.03em]"
          style={{ fontSize: size, color }}
        >
          {sign}
          {empty ? "0" : value}
        </span>
        <span className="text-faint text-lg font-semibold">Br</span>
      </div>
      {approx && (
        <div className="text-faint mt-1 font-mono text-[13px] font-medium">
          ≈ {approx}
        </div>
      )}
    </div>
  )
}
