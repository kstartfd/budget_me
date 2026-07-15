"use client"

import { useQuery } from "@tanstack/react-query"

import { todayISO } from "@/lib/month"

import { fetchSettings } from "./api"

/**
 * Read hook for the settings feature. The object is small and rarely changes,
 * so it uses a long staleTime; writes update the cache directly (see
 * mutations.ts).
 */

export const settingsKeys = {
  all: ["settings"] as const,
  detail: () => [...settingsKeys.all, "detail"] as const,
}

const settingsStaleMs = 10 * 60 * 1000

export function useSettings() {
  return useQuery({
    queryKey: settingsKeys.detail(),
    queryFn: fetchSettings,
    staleTime: settingsStaleMs,
  })
}

/**
 * "Today" (`YYYY-MM-DD`) that all current-period math runs on — the user's local
 * date, not the server's. The backend is UTC-only by design and cannot know the
 * user's zone, so at 01:00 Minsk its date is still yesterday; deriving the
 * current period from that would put the first hours of every day (and the 1st
 * of every month) in the wrong bucket.
 */
export function useToday(): string {
  return todayISO()
}
