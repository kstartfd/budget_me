"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"

/**
 * The app's front door. The bot's «Learn» button (and the menu button) open the
 * bare domain, so this is where a Mini App launch lands. Everyone goes to the
 * overview, which is the learner's home: it shows where to continue, or how to
 * set up a first language.
 *
 * It renders under the AuthGate, which already waited for the Telegram runtime
 * to settle, so the splash covers the redirect.
 */
export default function Home() {
  const router = useRouter()

  useEffect(() => {
    router.replace("/learning/overview")
  }, [router])

  return null
}
