"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/features/auth"
import * as api from "./api"

/** Everything the path tab reads and writes, keyed per learner. */
export function usePath(language: string, nativeLanguage: string) {
  const { user } = useAuth()
  const client = useQueryClient()
  const key = ["path", user?.id] as const
  const refresh = () => client.invalidateQueries({ queryKey: key })

  const active = useQuery({
    queryKey: [...key, "active", language],
    queryFn: () => api.fetchActivePath(language),
    enabled: !!user && !!language,
  })

  const history = useQuery({
    queryKey: [...key, "history"],
    queryFn: api.fetchHistory,
    enabled: !!user,
  })

  const startPlacement = useMutation({
    mutationFn: () => api.startPlacement(language, nativeLanguage),
    onSuccess: refresh,
  })

  const analyze = useMutation({
    mutationFn: api.analyzeTest,
    onSuccess: refresh,
  })

  const reset = useMutation({
    mutationFn: () => api.resetPath(language),
    onSuccess: refresh,
  })

  const startFinal = useMutation({
    mutationFn: () => api.startFinalTest(language, nativeLanguage),
    onSuccess: refresh,
  })

  return { active, history, startPlacement, analyze, reset, startFinal }
}

export function usePathTest(id: string | null) {
  return useQuery({
    queryKey: ["path", "test", id],
    queryFn: () => api.fetchTest(id!),
    enabled: !!id,
  })
}

export function useTopicDetail(topicId: string | null) {
  const { user } = useAuth()
  const client = useQueryClient()
  const key = ["path", "topic", topicId] as const

  const detail = useQuery({
    queryKey: key,
    queryFn: () => api.fetchTopicDetail(topicId!),
    enabled: !!user && !!topicId,
  })

  const generate = useMutation({
    mutationFn: (regenerate: boolean) =>
      api.generateExercises(topicId!, regenerate),
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  })

  const attempt = useMutation({
    mutationFn: ({ exerciseId, answer }: { exerciseId: string; answer: string }) =>
      api.submitAttempt(exerciseId, answer),
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  })

  const complete = useMutation({
    mutationFn: () => api.completeTopic(topicId!),
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  })

  return { detail, generate, attempt, complete }
}
