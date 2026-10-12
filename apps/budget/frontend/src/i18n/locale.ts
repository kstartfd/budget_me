"use server"

import { cookies, headers } from "next/headers"

import { defaultLocale, localeCookie, locales, type Locale } from "./config"

/**
 * Resolve the active locale for the current SSR request:
 * 1. honour the explicit cookie set by setUserLocale (user picked it),
 * 2. otherwise sniff `Accept-Language` and match against {@link locales}
 *    — this is the "force-pick the user's preferred language" path the
 *    product wants, so a Russian-speaker doesn't land on the English UI by
 *    accident,
 * 3. otherwise fall back to {@link defaultLocale} (EN — the global default).
 *
 * Note: this is intentionally separate from `users.country_code` resolution
 * (which is IP-based GeoIP on the backend). UI language and country are
 * different axes — a Russian-speaker in Germany still wants the Russian UI.
 */
export async function getUserLocale(): Promise<Locale> {
  const store = await cookies()
  const cookieValue = store.get(localeCookie)?.value
  if (isSupported(cookieValue)) return cookieValue

  const h = await headers()
  const accept = h.get("accept-language") ?? ""
  const sniffed = pickFromAcceptLanguage(accept)
  return sniffed ?? defaultLocale
}

export async function setUserLocale(locale: Locale): Promise<void> {
  const store = await cookies()
  store.set(localeCookie, locale, {
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
    sameSite: "lax",
  })
}

function isSupported(v: string | undefined): v is Locale {
  return !!v && (locales as readonly string[]).includes(v)
}

/** Returns the first supported language tag (`ru`/`en`) from a standard
 * Accept-Language header. Strips region/script subtags so `ru-RU` → `ru`. */
function pickFromAcceptLanguage(header: string): Locale | null {
  for (const raw of header.split(",")) {
    const tag = raw.trim().split(";")[0].trim().toLowerCase()
    if (!tag) continue
    const lang = tag.split("-")[0]
    if (isSupported(lang)) return lang
  }
  return null
}
