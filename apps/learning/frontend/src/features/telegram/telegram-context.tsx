"use client"

import { createContext, useContext, useEffect, useState } from "react"

import {
  getTelegramWebApp,
  type TelegramUser,
  type TelegramWebApp,
} from "@/lib/telegram"

interface TelegramContextValue {
  /** The WebApp handle, or null when not running inside Telegram. */
  webApp: TelegramWebApp | null
  /** The signed-in Telegram user, when available. */
  user: TelegramUser | null
  /**
   * True once we've settled whether Telegram is present — either the runtime
   * attached, or we've given up waiting for it. Guards against a flash of the
   * "open in Telegram" fallback while the script is still initializing.
   */
  ready: boolean
}

const TelegramContext = createContext<TelegramContextValue>({
  webApp: null,
  user: null,
  ready: false,
})

/**
 * Bridges the imperative Telegram WebApp runtime into React. On mount it waits
 * for `window.Telegram.WebApp` (loaded via the script in the root layout),
 * calls `ready()`/`expand()` so the Mini App fills its window, and exposes the
 * user + handle through context. Outside Telegram it settles as "ready, no
 * WebApp" so local browser dev keeps working.
 */
export function TelegramProvider({ children }: { children: React.ReactNode }) {
  const [value, setValue] = useState<TelegramContextValue>({
    webApp: null,
    user: null,
    ready: false,
  })

  useEffect(() => {
    let cancelled = false
    let attempts = 0

    function tryInit() {
      if (cancelled) return
      const wa = getTelegramWebApp()
      if (wa) {
        wa.ready()
        wa.expand()
        setValue({
          webApp: wa,
          user: wa.initDataUnsafe.user ?? null,
          ready: true,
        })
        return
      }
      // The script is loaded beforeInteractive, but poll briefly to cover any
      // race before window.Telegram attaches. After ~1s, settle as "no Telegram".
      attempts += 1
      if (attempts > 20) {
        setValue({ webApp: null, user: null, ready: true })
        return
      }
      window.setTimeout(tryInit, 50)
    }

    tryInit()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <TelegramContext.Provider value={value}>
      {children}
    </TelegramContext.Provider>
  )
}

/** Access the Telegram runtime (handle, user, ready flag) from any client component. */
export function useTelegram(): TelegramContextValue {
  return useContext(TelegramContext)
}
