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

/** A language the learner studies, with a line about its running program. */
export const pathTeacherSchema = z.object({
  id: z.string(),
  language: z.string(),
  translationLanguage: z.string(),
  createdAt: z.string(),
  path: z
    .object({
      id: z.string(),
      level: z.string(),
      topicsDone: z.number(),
      topicsTotal: z.number(),
    })
    .nullable(),
})
export type PathTeacher = z.infer<typeof pathTeacherSchema>

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
  startLevel: z.string().optional(),
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
  unit: z.string(),
  theory: z.string(),
  vocabulary: z.array(z.object({ word: z.string(), translation: z.string() })),
  status: z.enum(["not_started", "in_progress", "completed"]),
  completedAt: z.string().nullable(),
  /**
   * What the topic's current exercise set holds and how much of it is answered
   * right. A set that is not written yet is 0 of 0, and so is a topic read
   * where counts are not asked for.
   */
  exercises: z
    .object({ total: z.number(), done: z.number() })
    .default({ total: 0, done: 0 }),
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
  preferences: z
    .object({
      interests: z.string().optional(),
      variant: z.string().optional(),
      goal: z.string().optional(),
      minutesPerDay: z.number().optional(),
      daysPerWeek: z.number().optional(),
    })
    .default({}),
  topics: z.array(pathTopicSchema),
  createdAt: z.string(),
  archivedAt: z.string().nullable(),
})
export type Path = z.infer<typeof pathSchema>

/** The building of a program: it runs in the background on the learner's agent. */
export const pathJobSchema = z.object({
  id: z.string(),
  testId: z.string(),
  status: z.enum(["queued", "running", "done", "failed"]),
  error: z.string(),
  path: pathSchema.nullable(),
})
export type PathJob = z.infer<typeof pathJobSchema>

/** One guided exercise, including the rule and answer available before trying. */
export const pathExerciseSchema = z.object({
  id: z.string(),
  position: z.number(),
  kind: z.enum(["choice", "fill", "translate", "write", "listen", "speak"]),
  text: z.string(),
  prompt: z.string(),
  options: z.array(z.string()),
  answer: z.string(),
  rule: z.string(),
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

/** A rule of the rule book in a list or as a reference. */
export const pathRuleRefSchema = z.object({
  id: z.string(),
  level: z.string(),
  title: z.string(),
  unit: z.string(),
})
export type PathRuleRef = z.infer<typeof pathRuleRefSchema>

/** One rule: how it is built, examples, typical mistakes and references. */
export const pathRuleSchema = pathRuleRefSchema.extend({
  goal: z.string(),
  theory: z.string(),
  forms: z.array(z.string()),
  examples: z.array(z.object({ text: z.string(), translation: z.string() })),
  mistakes: z.array(z.object({ wrong: z.string(), right: z.string(), note: z.string() })),
  related: z.array(pathRuleRefSchema),
})
export type PathRule = z.infer<typeof pathRuleSchema>

/** A sentence the learner's agent wrote to show a rule. */
export const practiceExampleSchema = z.object({
  id: z.string(),
  text: z.string(),
  translation: z.string(),
})
export type PracticeExample = z.infer<typeof practiceExampleSchema>

/** The kinds of drill, after English Grammar in Use. */
export const practiceKinds = ["form", "choose", "correct", "make"] as const
export type PracticeKind = (typeof practiceKinds)[number]

/** One line of a drill, without its key: the key comes with a check. */
export const practiceItemSchema = z.object({
  id: z.string(),
  text: z.string(),
  hint: z.string(),
  options: z.array(z.string()),
})
export type PracticeItem = z.infer<typeof practiceItemSchema>

export const practiceBlockSchema = z.object({
  id: z.string(),
  kind: z.enum(practiceKinds),
  instruction: z.string(),
  items: z.array(practiceItemSchema),
})
export type PracticeBlock = z.infer<typeof practiceBlockSchema>

/** What the learner has on a rule right now: it goes when they leave its page. */
export const practiceSchema = z.object({
  examples: z.array(practiceExampleSchema),
  blocks: z.array(practiceBlockSchema),
})
export type Practice = z.infer<typeof practiceSchema>

/** The verdict on a line: whether the answer was right, and the key with its reason. */
export const practiceCheckSchema = z.object({
  correct: z.boolean(),
  key: z.string(),
  note: z.string(),
})
export type PracticeCheck = z.infer<typeof practiceCheckSchema>

export const pathTopicDetailSchema = z.object({
  path: pathSchema,
  topic: pathTopicSchema,
  exercises: z.array(pathExerciseSchema),
  attempts: z.array(pathAttemptSchema),
  rule: pathRuleSchema.nullish(),
})
export type PathTopicDetail = z.infer<typeof pathTopicDetailSchema>

/** A timed lesson on one topic. The clock is the server's: count down to endsAt. */
export const pathLessonSchema = z.object({
  id: z.string(),
  topicId: z.string(),
  status: z.enum(["active", "finished"]),
  startedAt: z.string(),
  endsAt: z.string(),
  finishedAt: z.string().nullable(),
  pausedAt: z.string().nullable(),
  pausedSeconds: z.number(),
  pauseCount: z.number(),
  attempts: z.number(),
  correct: z.number(),
})
export type PathLesson = z.infer<typeof pathLessonSchema>

/** The survey answered before the program is built; every part is optional. */
export interface Preferences {
  interests?: string
  variant?: string
  goal?: string
  minutesPerDay?: number
  daysPerWeek?: number
}
