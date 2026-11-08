import { z } from "zod"

/** One placement-test question as the test-taker sees it — never the answer. */
export const pathQuestionSchema = z.object({
  key: z.string(),
  skill: z.string(),
  text: z.string(),
  prompt: z.string(),
  options: z.array(z.string()),
})
export type PathQuestion = z.infer<typeof pathQuestionSchema>

export const pathTestSchema = z.object({
  id: z.string(),
  kind: z.enum(["placement", "final"]),
  language: z.string(),
  translationLanguage: z.string(),
  level: z.string(),
  levelReached: z.string(),
  passed: z.boolean(),
  status: z.enum(["in_progress", "done"]),
  questions: z.array(pathQuestionSchema),
  asked: z.number(),
  correct: z.number(),
  needed: z.number(),
  passNeed: z.number(),
  createdAt: z.string(),
  completedAt: z.string().nullable(),
})
export type PathTest = z.infer<typeof pathTestSchema>

/** One step of the program: 3 statuses, not started → in progress → completed. */
export const pathTopicSchema = z.object({
  id: z.string(),
  position: z.number(),
  skill: z.string(),
  title: z.string(),
  goal: z.string(),
  status: z.enum(["not_started", "in_progress", "completed"]),
  completedAt: z.string().nullable(),
})
export type PathTopic = z.infer<typeof pathTopicSchema>

export const pathSchema = z.object({
  id: z.string(),
  language: z.string(),
  translationLanguage: z.string(),
  level: z.string(),
  summary: z.string(),
  status: z.enum(["active", "finished", "abandoned"]),
  source: z.string(),
  topics: z.array(pathTopicSchema),
  createdAt: z.string(),
  archivedAt: z.string().nullable(),
})
export type Path = z.infer<typeof pathSchema>

/** One exercise as the learner sees it — the answer never leaves the server. */
export const pathExerciseSchema = z.object({
  id: z.string(),
  position: z.number(),
  kind: z.enum(["choice", "fill", "translate"]),
  text: z.string(),
  prompt: z.string(),
  options: z.array(z.string()),
})
export type PathExercise = z.infer<typeof pathExerciseSchema>

export const pathAttemptSchema = z.object({
  id: z.string(),
  exerciseId: z.string(),
  answer: z.string(),
  correct: z.boolean(),
  feedback: z.string(),
  createdAt: z.string(),
})
export type PathAttempt = z.infer<typeof pathAttemptSchema>

export const pathTopicDetailSchema = z.object({
  path: pathSchema,
  topic: pathTopicSchema,
  exercises: z.array(pathExerciseSchema),
  attempts: z.array(pathAttemptSchema),
})
export type PathTopicDetail = z.infer<typeof pathTopicDetailSchema>
