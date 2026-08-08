"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import { updateSettings } from "./api"
import { settingsKeys } from "./queries"
import type { Settings } from "./schemas"

/**
 * Write hook. Updates optimistically so the wheel and currency selector feel
 * instant (the picker would otherwise snap back for the mock latency window),
 * rolling back on error and reconciling with the server's echo on success.
 *
 * Note it does NOT touch the budget cache: month-start changes shift the
 * billing period, so callers that care (the settings page) invalidate the
 * budget queries themselves — keeping the feature dependency one-directional
 * (budget reads settings, never the reverse).
 */
export function useUpdateSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: updateSettings,
    onMutate: async (patch) => {
      await queryClient.cancelQueries({ queryKey: settingsKeys.detail() })
      const previous = queryClient.getQueryData<Settings>(settingsKeys.detail())
      if (previous) {
        // `resetHistory` is a confirmation for the server, not a setting — it
        // has no place in the cached Settings, and only the server's echo can
        // say what the reset actually did (`historyResetAt`).
        const settingsPatch = { ...patch }
        delete settingsPatch.resetHistory

        queryClient.setQueryData<Settings>(settingsKeys.detail(), {
          ...previous,
          ...settingsPatch,
        })
      }
      return { previous }
    },
    onError: (_error, _patch, context) => {
      if (context?.previous) {
        queryClient.setQueryData(settingsKeys.detail(), context.previous)
      }
    },
    onSuccess: (data) => {
      queryClient.setQueryData(settingsKeys.detail(), data)
    },
  })
}
