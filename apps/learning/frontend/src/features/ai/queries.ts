"use client"

import { useQuery } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import { fetchAiSettings } from "./api"

export function aiSettingsKey(userId: string | undefined) {
  return ["ai", userId, "settings"] as const
}

/** `data` is `null` while nothing has been configured yet. */
export function useAiSettings() {
  const { user } = useAuth()

  return useQuery({
    queryKey: aiSettingsKey(user?.id),
    queryFn: fetchAiSettings,
    enabled: !!user,
  })
}
