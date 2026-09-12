"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import { createTask, deleteTask, postponeTask, setTaskDone, updateTask } from "./api"
import { taskKeys } from "./queries"
import type { Task } from "./schemas"

/**
 * Write hooks. Ticking is optimistic — a checkbox that waits for a round trip
 * before filling in feels broken — while the rest just invalidate the feature
 * prefix, because they move a task between days and there is no cheap local
 * answer for what the other day now looks like.
 */

function useInvalidateTasks() {
  const queryClient = useQueryClient()

  return () => queryClient.invalidateQueries({ queryKey: taskKeys.all })
}

export function useCreateTask() {
  const invalidate = useInvalidateTasks()

  return useMutation({ mutationFn: createTask, onSuccess: invalidate })
}

export function useToggleTask(date: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { id: string; done: boolean }) =>
      setTaskDone({ ...input, date }),
    onMutate: async ({ id, done }) => {
      const key = taskKeys.day(date)
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Task[]>(key)

      queryClient.setQueryData<Task[]>(key, (rows) =>
        rows?.map((t) => (t.id === id ? { ...t, doneOn: done ? date : null } : t)),
      )

      return { previous }
    },
    onError: (_error, _input, context) => {
      if (context?.previous) {
        queryClient.setQueryData(taskKeys.day(date), context.previous)
      }
    },
    // Не инвалидируем на успех: строка уже в нужном состоянии, а перезапрос
    // на каждый чек делает список дёрганым.
  })
}

export function usePostponeTask() {
  const invalidate = useInvalidateTasks()

  return useMutation({ mutationFn: postponeTask, onSuccess: invalidate })
}

export function useUpdateTask() {
  const invalidate = useInvalidateTasks()

  return useMutation({ mutationFn: updateTask, onSuccess: invalidate })
}

export function useDeleteTask() {
  const invalidate = useInvalidateTasks()

  return useMutation({ mutationFn: deleteTask, onSuccess: invalidate })
}
