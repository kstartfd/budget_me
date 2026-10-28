import { z } from "zod"

/** Mirrors the backend's cap on one question. */
export const maxQuestionLength = 2000

export const chatMessageSchema = z.object({
  id: z.number(),
  role: z.enum(["user", "assistant"]),
  content: z.string(),
  createdAt: z.string(),
})
export type ChatMessage = z.infer<typeof chatMessageSchema>

/**
 * What one chat may spend with the agent in the current window: tokens used of
 * the limit. `resetsAt` is when the window ends; null while nothing is spent.
 */
export const budgetSchema = z.object({
  limit: z.number(),
  used: z.number(),
  resetsAt: z.string().nullable(),
})
export type Budget = z.infer<typeof budgetSchema>

/**
 * `available` is false while no agent is chosen or its provider has no chat;
 * the chat is then not offered at all (and has no budget).
 */
export const chatSchema = z.object({
  available: z.boolean(),
  /** The agent teaches the language and corrects mistakes, instead of just answering. */
  teacher: z.boolean(),
  messages: chatMessageSchema.array(),
  budget: budgetSchema.nullable(),
  /** The user's spending today across all their chats; same shape as a budget. */
  daily: budgetSchema.nullable(),
})
export type Chat = z.infer<typeof chatSchema>

/** The stored question and answer, sent once the answer arrived whole. */
export const chatDoneSchema = z.object({
  question: chatMessageSchema,
  answer: chatMessageSchema,
  budget: budgetSchema,
  daily: budgetSchema,
})
export type ChatDone = z.infer<typeof chatDoneSchema>

export const chatDeltaSchema = z.object({ text: z.string() })

export const chatStreamErrorSchema = z.object({
  code: z.number(),
  message: z.string(),
})
