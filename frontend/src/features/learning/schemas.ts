import { z } from "zod"

export const topicInputSchema = z.object({
  name: z.string().trim().min(1).max(80),
  language: z.string().trim().min(2).max(35),
  translationLanguage: z.string().trim().min(2).max(35),
})
export const topicSchema = topicInputSchema.extend({ id: z.string().uuid() })
export type Topic = z.infer<typeof topicSchema>
export type TopicInput = z.infer<typeof topicInputSchema>

export const entryInputSchema = z.object({
  topicId: z.string().uuid(),
  kind: z.enum(["word", "phrase"]),
  text: z.string().trim().min(1).max(1000),
  transcription: z.string().trim().max(4000),
  translation: z.string().trim().max(4000),
  nativeUsage: z.string().trim().max(4000),
  example: z.string().trim().max(4000),
  rule: z.string().trim().max(4000),
  status: z.enum(["inbox", "learning", "known"]),
})
export const entrySchema = entryInputSchema.extend({
  id: z.string().uuid(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type Entry = z.infer<typeof entrySchema>
export type EntryInput = z.infer<typeof entryInputSchema>

export const tokenInputSchema = z.object({
  name: z.string().trim().min(1).max(80),
  scope: z.enum(["capture", "read_write"]),
})
export const tokenSchema = tokenInputSchema.extend({
  id: z.string().uuid(),
  prefix: z.string(),
  createdAt: z.string(),
  lastUsedAt: z.string().nullable(),
})
export const issuedTokenSchema = tokenSchema.extend({ token: z.string() })
export type LearningToken = z.infer<typeof tokenSchema>
export type TokenInput = z.infer<typeof tokenInputSchema>
