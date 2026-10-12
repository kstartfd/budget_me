/**
 * Month-key helpers. A month key is `YYYY-MM` — the unit the whole app
 * budgets in. Keys are derived from local calendar dates (a Telegram user
 * thinks in their wall-clock month, not UTC).
 *
 * Kept as plain strings here so lib/ stays independent of feature modules;
 * the branded zod type lives in features/budget/schemas.ts.
 */
type MonthKey = string

export function monthKeyOf(date: Date): MonthKey {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`
}

/** Month key (`YYYY-MM`) of the server-provided `today` (`YYYY-MM-DD`). */
export function currentMonthKey(today: string): MonthKey {
  return today.slice(0, 7)
}

/**
 * The client's local calendar date as `YYYY-MM-DD`.
 *
 * The authoritative "today" comes from the server (see settings' `useToday` /
 * `readToday`); this is only the fallback used until that value loads, and the
 * mock's stand-in for it. Every other helper here takes `today` as a parameter
 * rather than reading the device clock, which is untrusted.
 */
export function todayISO(): string {
  const d = new Date()
  return `${monthKeyOf(d)}-${String(d.getDate()).padStart(2, "0")}`
}

/** Current wall-clock time as `HH:MM`. */
export function nowTime(): string {
  const d = new Date()
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`
}

export function shiftMonth(key: MonthKey, delta: number): MonthKey {
  const [y, m] = key.split("-").map(Number)
  const d = new Date(y, m - 1 + delta, 1)
  return monthKeyOf(d)
}

export function daysInMonth(key: MonthKey): number {
  const [y, m] = key.split("-").map(Number)
  return new Date(y, m, 0).getDate()
}

/** Clamp a start-of-period day into a real day of `key` (29–31 → month length). */
function clampStartDay(startDay: number, key: MonthKey): number {
  return Math.min(Math.max(Math.trunc(startDay), 1), daysInMonth(key))
}

/**
 * First calendar date (`YYYY-MM-DD`) of the billing period that opens in `key`,
 * given the day the period starts on. With the default `startDay` of 1 this is
 * `${key}-01`, so nothing shifts until the user changes the setting.
 */
export function periodStartISO(key: MonthKey, startDay: number): string {
  return `${key}-${String(clampStartDay(startDay, key)).padStart(2, "0")}`
}

/**
 * Last calendar date (`YYYY-MM-DD`) of that billing period: the day before the
 * next month's period start. With `startDay` 1 this is the last day of `key`.
 */
export function periodEndISO(key: MonthKey, startDay: number): string {
  const next = shiftMonth(key, 1)
  const [y, m] = next.split("-").map(Number)
  const end = new Date(y, m - 1, clampStartDay(startDay, next) - 1)
  return `${monthKeyOf(end)}-${String(end.getDate()).padStart(2, "0")}`
}

/**
 * Month key whose billing period contains `today`. Before this month's start
 * day, `today` still belongs to the period that opened last month (with the
 * default `startDay` of 1 this is always the current month, so nothing shifts).
 */
export function periodOpenKey(today: string, startDay: number): MonthKey {
  const key = currentMonthKey(today)
  const day = Number(today.slice(8, 10))
  return day >= clampStartDay(startDay, key) ? key : shiftMonth(key, -1)
}

/**
 * "5 июл — 4 авг" / "Jul 5 — Aug 4": the billing period *containing* `today` as
 * a compact label, honouring `startDay`.
 */
export function periodLabel(
  today: string,
  startDay: number,
  locale: string,
): string {
  const key = periodOpenKey(today, startDay)
  const fmt = new Intl.DateTimeFormat(locale, { day: "numeric", month: "short" })
  const at = (iso: string) => {
    const [y, m, d] = iso.split("-").map(Number)
    return fmt.format(new Date(y, m - 1, d)).replace(".", "")
  }
  return `${at(periodStartISO(key, startDay))} — ${at(periodEndISO(key, startDay))}`
}

/**
 * The [from, to] calendar range (`YYYY-MM-DD`, both inclusive) that `month`'s
 * billing period actually spans — the single place this gets decided, so a
 * fetch and a link built for the same `month` never disagree. The RUNNING
 * period (the one containing `today`, found via periodOpenKey) is capped at
 * `today`; any other (closed) period runs to its full end.
 */
export function periodBoundsISO(
  month: MonthKey,
  today: string,
  startDay: number,
): { from: string; to: string } {
  return month === currentMonthKey(today)
    ? { from: periodStartISO(periodOpenKey(today, startDay), startDay), to: today }
    : { from: periodStartISO(month, startDay), to: periodEndISO(month, startDay) }
}

/**
 * Days left in `key` counting `today`, or null when `key` isn't `today`'s
 * month. `today` is the server calendar date (`YYYY-MM-DD`).
 */
export function daysLeftInMonth(key: MonthKey, today: string): number | null {
  if (key !== currentMonthKey(today)) return null
  return daysInMonth(key) - Number(today.slice(8, 10)) + 1
}

/**
 * Days left (counting `today`) in the billing period that *contains* `today`,
 * honouring `startDay`. With the default `startDay` of 1 the period is the plain
 * calendar month, so this matches `daysLeftInMonth` — but once the user shifts
 * the start day (e.g. payday on the 5th) this counts to the period end (4th of
 * next month), keeping the dashboard's "Дней" consistent with the money figures
 * that already run over the period. `today` is the server calendar date.
 */
export function daysLeftInPeriod(today: string, startDay: number): number {
  const end = periodEndISO(periodOpenKey(today, startDay), startDay)
  const [ty, tm, td] = today.split("-").map(Number)
  const [ey, em, ed] = end.split("-").map(Number)
  const from = new Date(ty, tm - 1, td)
  const to = new Date(ey, em - 1, ed)

  return Math.round((to.getTime() - from.getTime()) / 86400000) + 1
}

/** "Июль 2026" / "July 2026" — capitalized month + year, no "г." suffix. */
export function monthTitle(key: MonthKey, locale: string): string {
  const [y, m] = key.split("-").map(Number)
  const name = new Intl.DateTimeFormat(locale, { month: "long" }).format(
    new Date(y, m - 1, 1),
  )
  return name.charAt(0).toUpperCase() + name.slice(1) + " " + y
}

/** Short month label for trend bars, e.g. "июл" / "Jul". */
export function monthShort(key: MonthKey, locale: string): string {
  const [y, m] = key.split("-").map(Number)
  return new Intl.DateTimeFormat(locale, { month: "short" })
    .format(new Date(y, m - 1, 1))
    .replace(".", "")
}

/**
 * Day heading for transaction groups: Today / Yesterday / "4 июля". `today` is
 * the server calendar date (`YYYY-MM-DD`) so "Today" tracks the server clock.
 */
export function dayLabel(
  iso: string,
  locale: string,
  todayText: string,
  yesterdayText: string,
  today: string,
): string {
  if (iso === today) return todayText
  const [y, m, d] = today.split("-").map(Number)
  const yest = new Date(y, m - 1, d - 1)
  if (iso === `${monthKeyOf(yest)}-${String(yest.getDate()).padStart(2, "0")}`)
    return yesterdayText
  const [yy, mm, dd] = iso.split("-").map(Number)
  return new Intl.DateTimeFormat(locale, { day: "numeric", month: "long" }).format(
    new Date(yy, mm - 1, dd),
  )
}

/** `iso` shifted by whole days, still `YYYY-MM-DD` (month/year roll over). */
export function shiftDate(iso: string, delta: number): string {
  const [y, m, d] = iso.split("-").map(Number)
  const at = new Date(y, m - 1, d + delta)
  return `${monthKeyOf(at)}-${String(at.getDate()).padStart(2, "0")}`
}

/** Whole days from `from` to `to` (negative when `to` is earlier). */
export function daysBetween(from: string, to: string): number {
  const [fy, fm, fd] = from.split("-").map(Number)
  const [ty, tm, td] = to.split("-").map(Number)
  return Math.round(
    (new Date(ty, tm - 1, td).getTime() - new Date(fy, fm - 1, fd).getTime()) /
      86400000,
  )
}

/** "12 сентября, сб" / "12 September, Sat" — one day as a screen subtitle. */
export function dateTitle(iso: string, locale: string): string {
  const [y, m, d] = iso.split("-").map(Number)
  return new Intl.DateTimeFormat(locale, {
    day: "numeric",
    month: "long",
    weekday: "short",
  })
    .format(new Date(y, m - 1, d))
    .replace(/\.$/, "")
}
