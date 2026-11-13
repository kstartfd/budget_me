"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"

import { useTelegram } from "@/features/telegram/telegram-context"

/**
 * The app's front door. The bot's «Learn» button (and the menu button) open the
 * bare domain, so this is where a Mini App launch lands: inside Telegram it goes
 * straight to the Teacher, in a plain browser to the topics list as before.
 *
 * It renders under the AuthGate, which already waited for the Telegram runtime
 * to settle, so `webApp` is final here and the splash covers the redirect.
 * `initData` is non-empty only in a real Telegram client (same check the session
 * boot uses), so a bare `window.Telegram` stub in a browser doesn't count.
 */
export default function Home() {
  const router = useRouter()
  const { webApp } = useTelegram()
  const inTelegram = (webApp?.initData ?? "").length > 0

  useEffect(() => {
    router.replace(inTelegram ? "/learning/path" : "/learning")
  }, [router, inTelegram])

  return null
}
