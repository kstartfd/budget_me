import { z } from "zod"

/**
 * Domain schemas for the settings feature. These mirror the payload shapes a
 * real `GET /settings` / `PUT /settings` endpoint will serve; while the app
 * runs on the mock data layer they are also the storage shape, so swapping the
 * fetchers later doesn't touch the hooks or components.
 */

/** ISO 4217 codes the app can render balances in. Br (BYN) is the default. */
export const currencySchema = z.enum(["BYN", "USD", "EUR", "RUB"])
export type Currency = z.infer<typeof currencySchema>

/** Display metadata for the currency selector (code + the symbol screens show). */
export const CURRENCIES: ReadonlyArray<{ code: Currency; symbol: string }> = [
  { code: "BYN", symbol: "Br" },
  { code: "USD", symbol: "$" },
  { code: "EUR", symbol: "€" },
  { code: "RUB", symbol: "₽" },
]

/**
 * Per-user preferences plus the server's authoritative `today`.
 *
 * `monthStartDay` is capped at 28: days 29–31 don't exist in every month, so
 * the picker offers 1–28 and the period helpers clamp anything larger to the
 * month's length.
 *
 * `today` (`YYYY-MM-DD`) is the server clock's calendar date — the app derives
 * "current month / period / days-left" from it instead of the untrusted device
 * clock. It is server-derived and read-only: never persisted, never sent on a
 * PUT (see `updateSettingsInputSchema`).
 */
export const settingsSchema = z.object({
  monthStartDay: z.number().int().min(1).max(28),
  currency: currencySchema,
  today: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
})
export type Settings = z.infer<typeof settingsSchema>

/**
 * PUT /settings accepts any subset of the editable fields (never the
 * server-derived `today`).
 */
export const updateSettingsInputSchema = settingsSchema
  .omit({ today: true })
  .partial()
export type UpdateSettingsInput = z.infer<typeof updateSettingsInputSchema>

/**
 * Fallback constants for the synchronous readers before the first fetch
 * resolves (calendar month, Belarusian ruble).
 */
export const DEFAULT_SETTINGS: Omit<Settings, "today"> = {
  monthStartDay: 1,
  currency: "BYN",
}
