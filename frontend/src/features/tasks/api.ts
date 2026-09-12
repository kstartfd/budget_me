import { mockTasks } from "./mock-db"
import {
  createTaskInputSchema,
  taskSchema,
  updateTaskInputSchema,
} from "./schemas"
import type { CreateTaskInput, Task, UpdateTaskInput } from "./schemas"

/**
 * Tasks API — the typed fetcher surface the query/mutation hooks call. Runs on
 * the mock store for now; the shapes below are exactly what `GET/POST/PATCH/
 * DELETE /api/v1/tasks` will answer, so connecting the backend is a change to
 * this file alone.
 *
 * THE ONE RULE WORTH KNOWING: a day's list is not stored, it is derived —
 * `plannedFor <= date AND doneOn IS NULL` plus whatever was ticked ON that
 * date. Undone work therefore carries itself forward with no nightly job and
 * no rows being rewritten at midnight, and "«Сдать анализы» висит 3-й день"
 * falls straight out of `date - plannedFor`.
 */

const latencyMs = 180

function settle<T>(value: T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), latencyMs))
}

/** Everything that belongs on `date`: still-open work, plus that day's done. */
export async function fetchDayTasks(date: string): Promise<Task[]> {
  const rows = mockTasks
    .all()
    .filter((t) => (t.doneOn === null ? t.plannedFor <= date : t.doneOn === date))
    .sort((a, b) => a.plannedFor.localeCompare(b.plannedFor) || a.createdAt.localeCompare(b.createdAt))

  return settle(rows.map((t) => taskSchema.parse(t)))
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  const parsed = createTaskInputSchema.parse(input)
  const task: Task = {
    id: `t-${Date.now().toString(36)}`,
    title: parsed.title,
    note: parsed.note && parsed.note.length > 0 ? parsed.note : null,
    plannedFor: parsed.plannedFor,
    doneOn: null,
    createdAt: new Date().toISOString(),
  }

  mockTasks.save([...mockTasks.all(), task])

  return settle(task)
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
  return settle(
    mutate(input.id, (t) => ({ ...t, doneOn: input.done ? input.date : null })),
  )
}

/** Move an open task to another day — the explicit «отложить», not the carry-over. */
export async function postponeTask(input: {
  id: string
  to: string
}): Promise<Task> {
  return settle(mutate(input.id, (t) => ({ ...t, plannedFor: input.to })))
}

export async function updateTask(input: UpdateTaskInput): Promise<Task> {
  const parsed = updateTaskInputSchema.parse(input)

  return settle(
    mutate(parsed.id, (t) => ({
      ...t,
      title: parsed.title ?? t.title,
      note: parsed.note === undefined ? t.note : parsed.note,
      plannedFor: parsed.plannedFor ?? t.plannedFor,
    })),
  )
}

export async function deleteTask(id: string): Promise<{ id: string }> {
  mockTasks.save(mockTasks.all().filter((t) => t.id !== id))

  return settle({ id })
}

function mutate(id: string, patch: (task: Task) => Task): Task {
  const rows = mockTasks.all()
  const found = rows.find((t) => t.id === id)
  if (!found) throw new Error("task not found")

  const next = taskSchema.parse(patch(found))
  mockTasks.save(rows.map((t) => (t.id === id ? next : t)))

  return next
}
