import { readToken } from "@/features/auth"
import { todayISO } from "@/lib/month"
import { browserTimezone } from "@/lib/time"

import { settingsSchema, updateSettingsInputSchema } from "./schemas"
import type { Settings, UpdateSettingsInput } from "./schemas"

/**
 * Settings API — the typed fetcher surface the query/mutation hooks call, live
 * against the Go backend (same-origin /api/v1, proxied by Next rewrites).
 * `GET/PUT /settings` are protected, so every request carries the session
 * Bearer token. Responses use the standard envelope `{code, message, data}`.
 *
 * Its own small `call()`/`ApiError` (mirroring features/auth/api.ts) rather than
 * reusing budget's: budget already imports this module's `readMonthStartDay`, so
 * pulling budget's `call` back in would form an import cycle.
 */

export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: number,
    readonly status: number,
  ) {
    super(message)
  }
}

/**
 * Last settings the server confirmed (fetch or update). Seeds the synchronous
 * readers below so `periodRange` in the budget api can build its date range
 * without awaiting a hook. Null until the first fetch resolves — the readers
 * fall back to safe defaults meanwhile.
 */
let current: Settings | null = null

async function call<T>(
  path: string,
  init: RequestInit,
  parse: (data: unknown) => T,
): Promise<T> {
  const token = readToken()
  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })
  const body = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!body || body.code !== 0) {
    throw new ApiError(
      body?.message ?? res.statusText,
      body?.code ?? -1,
      res.status,
    )
  }
  return parse(body.data)
}

/** Current user's settings. `GET /settings` → settingsSchema.parse. */
/**
 * Load the settings — and tell the backend which zone we are in. Requests carry
 * their own zone (the backend stores none for its own maths), but the nightly
 * worker runs with no client attached and still has to know which instants make
 * up "this month" for this user. This is where it learns.
 */
export async function fetchSettings(): Promise<Settings> {
  const tz = encodeURIComponent(browserTimezone())
  const settings = await call(`/settings?tz=${tz}`, { method: "GET" }, (d) =>
    settingsSchema.parse(d),
  )
  current = settings
  return settings
}

/**
 * Persist a partial update. `PUT /settings` accepts any subset of the editable
 * fields and echoes the full updated settings. An out-of-range day or unknown
 * currency answers HTTP 400 (surfaced as an ApiError).
 */
export async function updateSettings(
  input: UpdateSettingsInput,
): Promise<Settings> {
  const patch = updateSettingsInputSchema.parse(input)
  const settings = await call(
    "/settings",
    { method: "PUT", body: JSON.stringify(patch) },
    (d) => settingsSchema.parse(d),
  )
  current = settings
  return settings
}

/**
 * Synchronous read of the billing-period start day for non-hook call sites (the
 * budget api builds its `from`/`to` range from it). Reads the module cache set
 * by the last successful fetch/update; falls back to 1 (plain calendar month)
 * until the first fetch resolves. Mirrors auth's `readToken`.
 */
export function readMonthStartDay(): number {
  return current?.monthStartDay ?? 1
}

/**
 * Synchronous read of "today" (`YYYY-MM-DD`) for non-hook call sites (the budget
 * api caps the running period at today). The user's LOCAL date: the backend is
 * UTC-only and does not know the zone, so its idea of today is wrong for the
 * first hours of the local day (01:00 Minsk is still yesterday in UTC).
 */
export function readToday(): string {
  return todayISO()
}
