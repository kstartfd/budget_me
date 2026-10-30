"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import { enqueueFound, scanPicture } from "./api"
import { shrinkImage } from "./image"
import type { Found } from "./schemas"

/** Reads a photo the learner picked or took. */
export function useScanPicture(topicId: string) {
  return useMutation({
    mutationFn: async (file: File) =>
      scanPicture(topicId, await shrinkImage(file)),
  })
}

/**
 * Saves the chosen items. It does not stop at the first failure: `failed` are
 * the items that did not go through, to be offered again, and the rest are in.
 */
export function useAddFound(topicId: string, source: string) {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: async (items: Found[]) => {
      const results = await Promise.allSettled(
        items.map((item) => enqueueFound(topicId, item, source)),
      )

      return {
        added: results.filter((r) => r.status === "fulfilled").length,
        failed: items.filter((_, i) => results[i].status === "rejected"),
      }
    },
    onSuccess: async () => {
      const queryKey = ["learning", user?.id, "entries"]

      await client.invalidateQueries({ queryKey })
      // The queue writes the cards a moment later.
      setTimeout(() => void client.invalidateQueries({ queryKey }), 4000)
    },
  })
}
