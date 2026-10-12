/**
 * Money rendering for integer minor units (kopecks of Br). Formats follow
 * the "Budget Mini App" design: nbsp thousand groups, comma decimals, always
 * two fraction digits ("1 050,00"), the "Br" unit rendered by the caller so
 * it can style it separately. USD approximation rounds to whole dollars.
 */

/** "142,80" / "1 050,00" — number part only, unit is up to the caller. */
export function formatBr(minor: number): string {
  const [int, frac] = (Math.abs(minor) / 100).toFixed(2).split(".")
  return int.replace(/\B(?=(\d{3})+(?!\d))/g, " ") + "," + frac
}

/** Symbols for the currencies an amount can be shown *next to* (BYN is the
 *  amounts' own currency, so it never converts — see `fx` in ui-context). */
const currencySymbol: Record<string, string> = {
  USD: "$",
  EUR: "€",
  RUB: "₽",
}

/**
 * "2,87" / "3,34/100" — the rate pill's number. A ruble costs ~0.03 Br, which
 * reads as noise, so cheap units are quoted per 100 the way exchange boards do.
 */
export function formatRate(rate: number): string {
  const per100 = rate < 0.1
  const shown = (per100 ? rate * 100 : rate).toFixed(2).replace(".", ",")

  return per100 ? `${shown}/100` : shown
}

/**
 * "$44" / "€38" / "1 200 ₽" — the companion amount for a Br sum at the given
 * Br-per-unit rate. Rounded to whole units: it is an approximation, and the
 * design shows it as "≈ …".
 */
export function formatFx(
  minor: number,
  rate: number,
  currency: string,
): string {
  const value = Math.round(Math.abs(minor) / 100 / rate)
  const symbol = currencySymbol[currency] ?? currency

  // The ruble sign trails the amount; $ and € lead it.
  return currency === "RUB"
    ? value.toLocaleString("ru-RU").replace(/ /g, " ") + " " + symbol
    : symbol + value
}
