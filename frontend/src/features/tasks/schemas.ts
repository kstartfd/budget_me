import { z } from "zod"

/**
 * A task is planned FOR a local calendar day and carries no clock time: the
 * checklist is about days, not appointments.
 *
 * `doneOn` is the day it was ticked (not a timestamp) — it is what «сделано
 * сегодня» is counted from, and the reason an undone task simply reappears
 * on the next day rather than being moved by anything.
 */
const isoDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "expected YYYY-MM-DD")

export const taskSchema = z.object({
  id: z.string(),
  title: z.string(),
  note: z.string().nullable(),
  plannedFor: isoDate,
  doneOn: isoDate.nullable(),
  createdAt: z.string(),
})

export type Task = z.infer<typeof taskSchema>

export const createTaskInputSchema = z.object({
  title: z.string().trim().min(1).max(200),
  note: z.string().trim().max(500).nullable(),
  plannedFor: isoDate,
})

export type CreateTaskInput = z.infer<typeof createTaskInputSchema>

/**
 * `PUT /tasks/:id` replaces title/note/plannedFor wholesale (no PATCH) — so
 * every field here is required, not optional. A postpone is just this same
 * shape with only plannedFor actually changing (see api.ts's postponeTask).
 */
export const updateTaskInputSchema = z.object({
  id: z.string(),
  title: z.string().trim().min(1).max(200),
  note: z.string().trim().max(500).nullable(),
  plannedFor: isoDate,
})

export type UpdateTaskInput = z.infer<typeof updateTaskInputSchema>
