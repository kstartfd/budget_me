"use client"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { CURRENCIES, type Currency } from "../schemas"

/**
 * Main-currency picker built on the shadcn Select primitive. Options render as
 * "code · symbol" (e.g. "BYN · Br") so the selected value reads the same in the
 * trigger and the list.
 */
export function CurrencySelect({
  value,
  onChange,
  label,
}: {
  value: Currency
  onChange: (currency: Currency) => void
  /** Accessible name for the trigger. */
  label: string
}) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as Currency)}>
      <SelectTrigger
        aria-label={label}
        className="w-[140px] font-mono text-[12.5px] font-medium"
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {CURRENCIES.map((c) => (
          <SelectItem key={c.code} value={c.code} className="font-mono">
            {c.code} · {c.symbol}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
