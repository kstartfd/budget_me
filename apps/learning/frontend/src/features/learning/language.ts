"use client"

import { useLocale } from "next-intl"

export function useLanguageName() {
  const locale = useLocale()

  return (code: string) => {
    try {
      return (
        new Intl.DisplayNames([locale], { type: "language" }).of(code) ?? code
      )
    } catch {
      return code
    }
  }
}

/** `en-US` → `EN`: the short code shown inside a topic's round badge. */
export function languageBadge(code: string) {
  return code.split("-")[0].slice(0, 3).toUpperCase()
}
