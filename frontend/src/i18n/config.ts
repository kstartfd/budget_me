export const locales = ["ru", "en"] as const

export type Locale = (typeof locales)[number]

// Global fallback when neither the explicit cookie nor the user's
// Accept-Language header point at a supported locale. English is the
// safer default for an international product.
export const defaultLocale: Locale = "en"

export const localeCookie = "NEXT_LOCALE"

export const localeNames: Record<Locale, string> = {
  ru: "Русский",
  en: "English",
}
