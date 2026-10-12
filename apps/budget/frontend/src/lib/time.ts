/**
 * The UTC boundary. The backend stores and speaks instants only (RFC3339, UTC)
 * and knows nothing about zones — deliberately: the browser is the only party
 * that reliably knows where the user is. So this module is where wall-clock
 * ("13 July, 16:30" as the user sees it) turns into an instant and back.
 *
 * The rule: everything crossing the wire is UTC; everything shown to the user is
 * local. Nothing in between guesses.
 */

/** The user's IANA zone, e.g. "Europe/Minsk". Sent to endpoints that must group
 *  instants into the user's days and months (trend, limit recommendations). */
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
  } catch {
    return "UTC"
  }
}

/** Local wall-clock (`2026-07-13`, `16:30`) → the instant it names, as RFC3339
 *  UTC. The Date constructor reads the parts in the browser's zone, which is
 *  exactly the zone the user typed them in. */
export function toInstant(date: string, time: string): string {
  const [y, m, d] = date.split("-").map(Number)
  const [hh, mm] = time.split(":").map(Number)

  return new Date(y, m - 1, d, hh, mm, 0, 0).toISOString()
}

/** An instant → the local wall-clock the user should see it as. */
export function fromInstant(instant: string): { date: string; time: string } {
  const at = new Date(instant)
  const pad = (n: number) => String(n).padStart(2, "0")

  return {
    date: `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}`,
    time: `${pad(at.getHours())}:${pad(at.getMinutes())}`,
  }
}

/** Local midnight opening `date` → instant. Start of a range. */
export function dayStartInstant(date: string): string {
  return toInstant(date, "00:00")
}

/**
 * Instant that ends the day `date` — local midnight of the NEXT day. Ranges are
 * half-open [from, to), so this includes everything logged on `date` itself.
 */
export function dayEndInstant(date: string): string {
  const [y, m, d] = date.split("-").map(Number)

  return new Date(y, m - 1, d + 1, 0, 0, 0, 0).toISOString()
}
