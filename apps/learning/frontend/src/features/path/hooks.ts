"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/features/auth"
import * as api from "./api"
import type { PathLesson } from "./schemas"

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
    mutationFn: (level?: string) => api.startPlacement(language, nativeLanguage, level),
    onSuccess: refresh,
  })

  const analyze = useMutation({
    mutationFn: (id: string) => api.analyzeTest(id),
    onSuccess: refresh,
  })

  const reset = useMutation({
    mutationFn: () => api.resetPath(language),
    onSuccess: refresh,
  })

  const startFinal = useMutation({
    mutationFn: (native?: string) => api.startFinalTest(language, native ?? nativeLanguage),
    onSuccess: refresh,
  })

  return { active, history, startPlacement, analyze, reset, startFinal }
}

/** The learner's teachers, and adding one. */
export function useTeachers() {
  const { user } = useAuth()
  const client = useQueryClient()

  const teachers = useQuery({
    queryKey: ["path", user?.id, "teachers"],
    queryFn: api.fetchTeachers,
    enabled: !!user,
  })

  const create = useMutation({
    mutationFn: (input: { language: string; translationLanguage: string }) =>
      api.createTeacher(input.language, input.translationLanguage),
    onSuccess: () => client.invalidateQueries({ queryKey: ["path", user?.id] }),
  })

  return { teachers, create }
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

  const refresh = () =>
    Promise.all([
      client.invalidateQueries({ queryKey: key }),
      client.invalidateQueries({ queryKey: ["path", user?.id] }),
    ])

  const generate = useMutation({
    mutationFn: (regenerate: boolean) =>
      api.generateExercises(topicId!, regenerate),
    onSuccess: refresh,
  })

  const complete = useMutation({
    mutationFn: () => api.completeTopic(topicId!),
    onSuccess: refresh,
  })

  return { detail, generate, complete }
}

export function useTopicExerciseAttempt(topicId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({
      exerciseId,
      answer,
    }: {
      exerciseId: string
      answer: string
    }) => api.submitAttempt(exerciseId, answer),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: ["path", "topic", topicId] }),
        client.invalidateQueries({ queryKey: ["path"] }),
      ]),
  })
}

/**
 * The lesson clock of a topic: the learner's running lesson (null when none is
 * running or it runs on another topic), the history of the topic's lessons and
 * the moves on the clock. Every move answers with the lesson as the server has it.
 */
export function useLesson(topicId: string) {
  const { user } = useAuth()
  const client = useQueryClient()
  const activeKey = ["path", "lesson", "active", user?.id] as const
  const listKey = ["path", "lessons", topicId] as const

  const active = useQuery({
    queryKey: activeKey,
    queryFn: api.fetchActiveLesson,
    enabled: !!user,
  })

  const history = useQuery({
    queryKey: listKey,
    queryFn: () => api.fetchLessons(topicId),
    enabled: !!user,
  })

  const settle = (lesson: PathLesson) => {
    client.setQueryData(
      activeKey,
      lesson.status === "active" ? lesson : null,
    )
    return client.invalidateQueries({ queryKey: listKey })
  }

  const start = useMutation({
    mutationFn: () => api.startLesson(topicId),
    onSuccess: async (lesson) => {
      await settle(lesson)
      await client.invalidateQueries({ queryKey: ["path"] })
    },
  })

  const change = useMutation({
    mutationFn: ({ id, action }: { id: string; action: "pause" | "resume" | "finish" }) =>
      api.changeLesson(id, action),
    onSuccess: settle,
  })

  const lesson = active.data?.topicId === topicId ? active.data : null

  return { lesson, history, start, change }
}
