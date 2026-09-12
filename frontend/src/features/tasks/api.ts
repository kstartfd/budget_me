import { readToken } from "@/features/auth"

import {
  createTaskInputSchema,
  taskSchema,
  updateTaskInputSchema,
} from "./schemas"
import type { CreateTaskInput, Task, UpdateTaskInput } from "./schemas"

/**
 * Tasks API — the typed fetcher surface the query/mutation hooks call, live
 * against the Go backend (same-origin /api/v1, proxied by Next rewrites).
 * Own small call()/ApiError, mirroring features/settings/api.ts — every
 * feature keeps its own rather than sharing one, so nothing here risks an
 * import cycle with a sibling feature.
 *
 * THE ONE RULE WORTH KNOWING: a day's list is not stored, it is derived —
 * `plannedFor <= date AND doneOn IS NULL` plus whatever was ticked ON that
 * date. Undone work therefore carries itself forward with no nightly job and
 * no rows being rewritten at midnight, and "«Сдать анализы» висит 3-й день"
 * falls straight out of `date - plannedFor` (see backend/internal/tasks).
 */

class ApiError extends Error {
  constructor(
    message: string,
    readonly code: number,
    readonly status: number,
  ) {
    super(message)
  }
}

async function call<T>(
  path: string,
  init: RequestInit,
  parse: (data: unknown) => T,
): Promise<T> {
  const token = readToken()
  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })
  const body = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!body || body.code !== 0) {
    throw new ApiError(
      body?.message ?? res.statusText,
      body?.code ?? -1,
      res.status,
    )
  }
  return parse(body.data)
}

/** Everything that belongs on `date`: still-open work, plus that day's done. */
export async function fetchDayTasks(date: string): Promise<Task[]> {
  return call(`/tasks?date=${date}`, { method: "GET" }, (d) =>
    taskSchema.array().parse(d),
  )
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  const parsed = createTaskInputSchema.parse(input)

  return call(
    "/tasks",
    { method: "POST", body: JSON.stringify(parsed) },
    (d) => taskSchema.parse(d),
  )
}

/**
 * Tick or untick. `date` is the day being looked at, not necessarily today —
 * ticking a task while browsing yesterday marks it done on yesterday, which is
 * what keeps that day's «сделано» honest.
 */
export async function setTaskDone(input: {
  id: string
  done: boolean
  date: string
}): Promise<Task> {
  return call(
    `/tasks/${input.id}/done`,
    { method: "PUT", body: JSON.stringify({ doneOn: input.done ? input.date : null }) },
    (d) => taskSchema.parse(d),
  )
}

/**
 * Move an open task to another day — the explicit «отложить», not the
 * carry-over. `PUT /tasks/:id` replaces title/note/plannedFor wholesale, so a
 * postpone still has to carry the task's current title/note through
 * unchanged — it is that same endpoint with only plannedFor actually moving.
 */
export async function postponeTask(input: {
  id: string
  title: string
  note: string | null
  to: string
}): Promise<Task> {
  return updateTask({
    id: input.id,
    title: input.title,
    note: input.note,
    plannedFor: input.to,
  })
}

export async function updateTask(input: UpdateTaskInput): Promise<Task> {
  const parsed = updateTaskInputSchema.parse(input)

  return call(
    `/tasks/${parsed.id}`,
    {
      method: "PUT",
      body: JSON.stringify({
        title: parsed.title,
        note: parsed.note,
        plannedFor: parsed.plannedFor,
      }),
    },
    (d) => taskSchema.parse(d),
  )
}

export async function deleteTask(id: string): Promise<{ id: string }> {
  return call(`/tasks/${id}`, { method: "DELETE" }, (d) => d as { id: string })
}
