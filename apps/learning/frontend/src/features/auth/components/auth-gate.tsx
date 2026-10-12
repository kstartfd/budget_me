"use client"

import { WalletLogoIcon } from "@/components/app-icons"

import { useAuth } from "../auth-context"
import { AuthScreen } from "./auth-screen"

/**
 * Blocks the app until a session exists: pulsing splash while the session
 * boots (stored token check / Telegram initData exchange), the login screen
 * for guests, the app for authenticated users.
 */
export function AuthGate({ children }: { children: React.ReactNode }) {
  const { status } = useAuth()

  if (status === "booting") {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <div
          aria-hidden
          className="bg-primary text-primary-foreground flex size-12 animate-pulse items-center justify-center rounded-2xl"
        >
          <WalletLogoIcon size={24} />
        </div>
      </main>
    )
  }

  if (status === "guest") return <AuthScreen />

  return <>{children}</>
}
