/**
 * Minimal typed access to the Telegram Mini App runtime — the global object
 * injected by `telegram.org/js/telegram-web-app.js` (loaded in the root layout).
 *
 * We wrap the raw `window.Telegram.WebApp` instead of pulling in an SDK so the
 * skeleton has zero extra dependencies. Swap in `@telegram-apps/sdk-react` later
 * if richer typing / signal-based reactivity is wanted — the app only depends on
 * the helpers below, so the blast radius is small.
 */

export interface TelegramUser {
  id: number
  first_name: string
  last_name?: string
  username?: string
  language_code?: string
  photo_url?: string
  is_premium?: boolean
}

export interface TelegramWebApp {
  initData: string
  initDataUnsafe: {
    user?: TelegramUser
    start_param?: string
    [key: string]: unknown
  }
  version: string
  platform: string
  colorScheme: "light" | "dark"
  themeParams: Record<string, string>
  isExpanded: boolean
  viewportHeight: number
  ready: () => void
  expand: () => void
  close: () => void
  setHeaderColor: (color: string) => void
  setBackgroundColor: (color: string) => void
  onEvent: (event: string, handler: () => void) => void
  offEvent: (event: string, handler: () => void) => void
  HapticFeedback?: {
    impactOccurred: (
      style: "light" | "medium" | "heavy" | "rigid" | "soft",
    ) => void
    notificationOccurred: (type: "error" | "success" | "warning") => void
    selectionChanged: () => void
  }
  MainButton: {
    text: string
    show: () => void
    hide: () => void
    setText: (text: string) => void
    onClick: (cb: () => void) => void
    offClick: (cb: () => void) => void
  }
  BackButton: {
    show: () => void
    hide: () => void
    onClick: (cb: () => void) => void
    offClick: (cb: () => void) => void
  }
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

/** The Telegram WebApp instance, or null when not running inside a Telegram client. */
export function getTelegramWebApp(): TelegramWebApp | null {
  if (typeof window === "undefined") return null
  return window.Telegram?.WebApp ?? null
}

/**
 * True when the page is actually rendered inside a Telegram client. `initData`
 * is only populated by real Telegram clients, so an empty string means we're in
 * a plain browser (e.g. `pnpm dev`).
 */
export function isTelegramEnv(): boolean {
  const wa = getTelegramWebApp()
  return !!wa && wa.initData.length > 0
}
