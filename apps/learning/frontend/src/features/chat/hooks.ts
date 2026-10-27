"use client"

import { useCallback, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import { ChatError, clearChat, fetchChat, streamChat } from "./api"
import type { Chat } from "./schemas"

export function chatKey(userId: string | undefined, entryId: string) {
  return ["learning", userId, "chat", entryId] as const
}

/** The conversation of an entry; `data.available` is false when there is no agent to ask. */
export function useChat(entryId: string) {
  const { user } = useAuth()

  return useQuery({
    queryKey: chatKey(user?.id, entryId),
    queryFn: () => fetchChat(entryId),
    enabled: !!user,
  })
}

/** The question being answered right now, and the answer as far as it has come. */
export interface ChatTurn {
  question: string
  answer: string
}

/**
 * Sends questions and streams the answer into `turn`. Once an answer is whole
 * it moves into the cached history. A failed turn leaves the history as it was
 * (the server stores nothing), so `send` resolves false and the caller keeps
 * the text for another try.
 */
export function useChatTurn(entryId: string) {
  const { user } = useAuth()
  const client = useQueryClient()
  const [turn, setTurn] = useState<ChatTurn | null>(null)
  const [error, setError] = useState<ChatError | null>(null)

  const send = useCallback(
    async (text: string): Promise<boolean> => {
      const key = chatKey(user?.id, entryId)

      setError(null)
      setTurn({ question: text, answer: "" })

      try {
        const done = await streamChat(entryId, text, (piece) =>
          setTurn(
            (current) =>
              current && { ...current, answer: current.answer + piece },
          ),
        )

        client.setQueryData<Chat>(
          key,
          (old) =>
            old && {
              ...old,
              messages: [...old.messages, done.question, done.answer],
              budget: done.budget,
            },
        )

        return true
      } catch (e) {
        setError(e instanceof ChatError ? e : new ChatError(String(e)))
        // The server keeps answering after a dropped connection, so the answer
        // may be stored although the stream failed.
        await client.invalidateQueries({ queryKey: key })

        return false
      } finally {
        setTurn(null)
      }
    },
    [client, entryId, user?.id],
  )

  return { turn, error, send, sending: turn !== null }
}

/** Forgets the whole conversation, on the server and in the cache. */
export function useClearChat(entryId: string) {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: () => clearChat(entryId),
    onSuccess: () =>
      client.setQueryData<Chat>(
        chatKey(user?.id, entryId),
        (old) => old && { ...old, messages: [] },
      ),
  })
}
