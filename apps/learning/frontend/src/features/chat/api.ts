import { readToken } from "@/features/auth"
import { call } from "@/features/learning/api"

import {
  chatDeltaSchema,
  chatDoneSchema,
  chatSchema,
  chatStreamErrorSchema,
  dialogSchema,
  transcriptSchema,
  type ChatDone,
} from "./schemas"

/** Business codes of the backend that the chat tells apart. */
export const chatErrorCode = {
  invalid: 10001,
  busy: 40001,
  /** Asking faster than allowed: wait a few seconds. */
  tooMany: 42901,
  /** The chat used up its tokens; `details[0]` is when its window ends. */
  budget: 42902,
  /** The user's (or their API key's) tokens for today are used up; resets at UTC midnight. */
  daily: 42903,
  unavailable: 50201,
} as const

/** A refused or failed turn; `code` is the backend's business code, when it sent one. */
export class ChatError extends Error {
  constructor(
    message: string,
    readonly code?: number,
    readonly details: string[] = [],
  ) {
    super(message)
    this.name = "ChatError"
  }
}

export const fetchChat = (entryId: string) =>
  call(`/entries/${entryId}/chat`, "GET", (d) => chatSchema.parse(d))

export const clearChat = (entryId: string) =>
  call(`/entries/${entryId}/chat`, "DELETE", () => undefined)

export const fetchTopicChat = (topicId: string) =>
  call(`/topics/${topicId}/chat`, "GET", (d) => chatSchema.parse(d))

/** "Clear chat": the open dialog is closed and kept in the history. */
export const archiveTopicChat = (topicId: string) =>
  call(`/topics/${topicId}/chat/archive`, "POST", () => undefined)

export const fetchDialogs = (topicId: string) =>
  call(`/topics/${topicId}/chat/history`, "GET", (d) =>
    dialogSchema.array().parse(d),
  )

export const fetchDialog = (id: string) =>
  call(`/chat-dialogs/${id}`, "GET", (d) => transcriptSchema.parse(d))

export const deleteDialog = (id: string) =>
  call(`/chat-dialogs/${id}`, "DELETE", () => undefined)

/** The chat of a topic of the learner's program. */
export const fetchPathTopicChat = (topicId: string) =>
  call(`/path/topics/${topicId}/chat`, "GET", (d) => chatSchema.parse(d))

export const clearPathTopicChat = (topicId: string) =>
  call(`/path/topics/${topicId}/chat`, "DELETE", () => undefined)

type StreamEvent = { name: string; data: unknown }

/** One server-sent event: `event: name` and `data: {json}` lines. */
function parseEvent(block: string): StreamEvent | null {
  let name = "message"
  let data = ""

  for (const line of block.split("\n")) {
    if (line.startsWith("event:")) name = line.slice(6).trim()
    else if (line.startsWith("data:")) data += line.slice(5).trim()
  }

  if (!data) return null

  try {
    return { name, data: JSON.parse(data) }
  } catch {
    return null
  }
}

/**
 * Asks one question and calls `onDelta` with each piece of the answer as it
 * arrives. Resolves with the stored messages once the answer is whole; any
 * failure rejects with a {@link ChatError}. A stream that breaks midway also
 * rejects, although the server may still store the answer: callers refetch.
 */
export const streamChat = (
  entryId: string,
  text: string,
  onDelta: (piece: string) => void,
) => streamAnswer(`/entries/${entryId}/chat`, text, onDelta)

/** The same for the free conversation with the teacher of a topic. */
export const streamTopicChat = (
  topicId: string,
  text: string,
  onDelta: (piece: string) => void,
) => streamAnswer(`/topics/${topicId}/chat`, text, onDelta)

export const streamPathTopicChat = (
  topicId: string,
  text: string,
  onDelta: (piece: string) => void,
) => streamAnswer(`/path/topics/${topicId}/chat`, text, onDelta)

async function streamAnswer(
  path: string,
  text: string,
  onDelta: (piece: string) => void,
): Promise<ChatDone> {
  const token = readToken()
  const res = await fetch(`/api/v1/learning${path}`, {
    method: "POST",
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ text }),
  })

  if (!res.ok || !res.body) {
    const body = await res.json().catch(() => null)
    throw new ChatError(
      body?.message ?? res.statusText,
      body?.code,
      Array.isArray(body?.details) ? body.details : [],
    )
  }

  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ""
  let done: ChatDone | null = null

  for (;;) {
    const { value, done: finished } = await reader.read()
    if (finished) break

    buffer += value.replace(/\r\n/g, "\n")

    for (
      let cut = buffer.indexOf("\n\n");
      cut >= 0;
      cut = buffer.indexOf("\n\n")
    ) {
      const event = parseEvent(buffer.slice(0, cut))
      buffer = buffer.slice(cut + 2)

      if (!event) continue

      if (event.name === "delta") {
        onDelta(chatDeltaSchema.parse(event.data).text)
      } else if (event.name === "done") {
        done = chatDoneSchema.parse(event.data)
      } else if (event.name === "error") {
        const failure = chatStreamErrorSchema.parse(event.data)
        throw new ChatError(failure.message, failure.code)
      }
    }
  }

  if (!done) throw new ChatError("The answer was interrupted")

  return done
}
