"use client"

import { useState } from "react"
import { QueryClientProvider } from "@tanstack/react-query"

import { TelegramProvider } from "@/features/telegram/telegram-context"
import { makeQueryClient } from "@/lib/query-client"

/**
 * Client-side app providers, mounted once by the root layout so both the React
 * Query cache and the Telegram runtime survive client navigation.
 *
 * The QueryClient is created in lazy state so it's stable across renders (a new
 * client per render would throw the cache away every time).
 */
export function AppProviders({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(makeQueryClient)
  return (
    <QueryClientProvider client={queryClient}>
      <TelegramProvider>{children}</TelegramProvider>
    </QueryClientProvider>
  )
}
