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
 * `available` is false while no agent is chosen or its provider has no chat;
 * the chat is then not offered at all.
 */
export const chatSchema = z.object({
  available: z.boolean(),
  messages: chatMessageSchema.array(),
})
export type Chat = z.infer<typeof chatSchema>

/** The stored question and answer, sent once the answer arrived whole. */
export const chatDoneSchema = z.object({
  question: chatMessageSchema,
  answer: chatMessageSchema,
})
export type ChatDone = z.infer<typeof chatDoneSchema>

export const chatDeltaSchema = z.object({ text: z.string() })

export const chatStreamErrorSchema = z.object({
  code: z.number(),
  message: z.string(),
})
