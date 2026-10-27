"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import { deleteAiSettings, saveAiSettings } from "./api"
import { aiSettingsKey } from "./queries"

/** The saved settings come back from the endpoint, so the cache takes them as-is. */
export function useSaveAiSettings() {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: saveAiSettings,
    onSuccess: (saved) => client.setQueryData(aiSettingsKey(user?.id), saved),
  })
}

/** Forgets the agent and its key on the server. */
export function useDeleteAiSettings() {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: deleteAiSettings,
    onSuccess: () => client.setQueryData(aiSettingsKey(user?.id), null),
  })
}
