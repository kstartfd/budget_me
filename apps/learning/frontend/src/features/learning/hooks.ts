"use client"

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { useAuth } from "@/features/auth"
import * as api from "./api"
import type { Entry } from "./schemas"

export function useLearning() {
  const { user } = useAuth()
  const client = useQueryClient()
  const key = ["learning", user?.id] as const
  const refresh = () => client.invalidateQueries({ queryKey: key })
  const topics = useQuery({
    queryKey: [...key, "topics"],
    queryFn: api.fetchTopics,
    enabled: !!user,
    refetchInterval: 15000,
  })
  const saveTopic = useMutation({
    mutationFn: api.saveTopic,
    onSuccess: refresh,
  })
  const deleteTopic = useMutation({
    mutationFn: api.deleteTopic,
    onSuccess: refresh,
  })
  const saveEntry = useMutation({
    mutationFn: api.saveEntry,
    onSuccess: refresh,
  })
  const deleteEntry = useMutation({
    mutationFn: api.deleteEntry,
    onSuccess: refresh,
  })
  const fillEntry = useMutation({
    mutationFn: api.fillEntry,
    onSuccess: refresh,
  })
  return { topics, saveTopic, deleteTopic, saveEntry, deleteEntry, fillEntry }
}

export function useEntries(
  topicId: string,
  filters: { kind: string; status: string; search: string },
) {
  const { user } = useAuth()
  return useInfiniteQuery({
    queryKey: ["learning", user?.id, "entries", topicId, filters],
    queryFn: ({ pageParam }) => api.fetchEntries(topicId, pageParam, filters),
    initialPageParam: 0,
    getNextPageParam: (last, pages) =>
      last.length === api.PAGE_SIZE ? pages.length * api.PAGE_SIZE : undefined,
    enabled: !!user && !!topicId,
    refetchInterval: 10000,
  })
}

export function useEntry(id: string | null, initialData?: Entry) {
  const { user } = useAuth()
  return useQuery({
    queryKey: ["learning", user?.id, "entry", id],
    queryFn: () => api.fetchEntry(id!),
    initialData,
    enabled: !!user && !!id,
    refetchInterval: 10000,
  })
}

export function useLearningTokens() {
  const { user } = useAuth()
  const client = useQueryClient()
  const key = ["learning", user?.id, "tokens"] as const
  const refresh = () => client.invalidateQueries({ queryKey: key })
  const tokens = useQuery({
    queryKey: key,
    queryFn: api.fetchTokens,
    enabled: !!user,
  })
  const create = useMutation({
    mutationFn: api.createToken,
    onSuccess: refresh,
    gcTime: 0,
  })
  const revoke = useMutation({
    mutationFn: api.revokeToken,
    onSuccess: refresh,
  })
  // A secret is fetched on demand and never kept: no cache entry outlives the call.
  const reveal = (id: string) =>
    client.fetchQuery({
      queryKey: [...key, id, "secret"],
      queryFn: () => api.fetchTokenSecret(id),
      staleTime: 0,
      gcTime: 0,
    })
  return { tokens, create, revoke, reveal }
}
