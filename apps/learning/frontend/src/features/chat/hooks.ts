"use client"

import { useCallback, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import {
  archiveTopicChat,
  ChatError,
  clearChat,
  clearPathTopicChat,
  deleteDialog,
  fetchChat,
  fetchDialog,
  fetchDialogs,
  fetchPathTopicChat,
  fetchTopicChat,
  streamChat,
  streamPathTopicChat,
  streamTopicChat,
} from "./api"
import type { Chat, ChatDone } from "./schemas"

export function chatKey(userId: string | undefined, entryId: string) {
  return ["learning", userId, "chat", entryId] as const
}

/** The open dialog of a topic; its history is a child of this key. */
export function topicChatKey(userId: string | undefined, topicId: string) {
  return ["learning", userId, "topic-chat", topicId] as const
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
function useTurn(
  key: readonly unknown[],
  stream: (text: string, onDelta: (piece: string) => void) => Promise<ChatDone>,
) {
  const client = useQueryClient()
  const [turn, setTurn] = useState<ChatTurn | null>(null)
  const [error, setError] = useState<ChatError | null>(null)

  const send = useCallback(
    async (text: string): Promise<boolean> => {
      setError(null)
      setTurn({ question: text, answer: "" })

      try {
        const done = await stream(text, (piece) =>
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
              daily: done.daily,
            },
        )

        return true
      } catch (e) {
        setError(e instanceof ChatError ? e : new ChatError(String(e)))
        // The server keeps answering after a dropped connection, so the answer
        // may be stored although the stream failed.
        await client.invalidateQueries({ queryKey: key, exact: true })

        return false
      } finally {
        setTurn(null)
      }
    },
    [client, key, stream],
  )

  return { turn, error, send, sending: turn !== null }
}

export function useChatTurn(entryId: string) {
  const { user } = useAuth()
  const key = useMemo(() => chatKey(user?.id, entryId), [user?.id, entryId])
  const stream = useCallback(
    (text: string, onDelta: (piece: string) => void) =>
      streamChat(entryId, text, onDelta),
    [entryId],
  )

  return useTurn(key, stream)
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

/** The open dialog with the teacher of a topic. */
export function useTopicChat(topicId: string | undefined) {
  const { user } = useAuth()

  return useQuery({
    queryKey: topicChatKey(user?.id, topicId ?? ""),
    queryFn: () => fetchTopicChat(topicId!),
    enabled: !!user && !!topicId,
  })
}

export function useTopicChatTurn(topicId: string) {
  const { user } = useAuth()
  const key = useMemo(
    () => topicChatKey(user?.id, topicId),
    [user?.id, topicId],
  )
  const stream = useCallback(
    (text: string, onDelta: (piece: string) => void) =>
      streamTopicChat(topicId, text, onDelta),
    [topicId],
  )

  return useTurn(key, stream)
}

/**
 * "Clear chat": the open dialog is closed and moves into the history, and the
 * next message starts a new one. The budget carries over, so it stays as it is.
 */
export function useArchiveTopicChat(topicId: string) {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: () => archiveTopicChat(topicId),
    onSuccess: async () => {
      client.setQueryData<Chat>(
        topicChatKey(user?.id, topicId),
        (old) => old && { ...old, messages: [] },
      )
      await client.invalidateQueries({
        queryKey: [...topicChatKey(user?.id, topicId), "history"],
      })
    },
  })
}

/** The closed dialogs of a topic, newest first. */
export function useDialogs(topicId: string, enabled = true) {
  const { user } = useAuth()

  return useQuery({
    queryKey: [...topicChatKey(user?.id, topicId), "history"],
    queryFn: () => fetchDialogs(topicId),
    enabled: !!user && enabled,
  })
}

/** One closed dialog with everything said in it. */
export function useDialog(id: string | null) {
  const { user } = useAuth()

  return useQuery({
    queryKey: ["learning", user?.id, "chat-dialog", id],
    queryFn: () => fetchDialog(id!),
    enabled: !!user && !!id,
  })
}

export function useDeleteDialog(topicId: string) {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: deleteDialog,
    onSuccess: () =>
      client.invalidateQueries({
        queryKey: [...topicChatKey(user?.id, topicId), "history"],
      }),
  })
}

export function pathTopicChatKey(userId: string | undefined, topicId: string) {
  return ["learning", userId, "path-topic-chat", topicId] as const
}

/** The chat of one topic of the program; always the teacher. */
export function usePathTopicChat(topicId: string) {
  const { user } = useAuth()

  return useQuery({
    queryKey: pathTopicChatKey(user?.id, topicId),
    queryFn: () => fetchPathTopicChat(topicId),
    enabled: !!user,
  })
}

export function usePathTopicChatTurn(topicId: string) {
  const { user } = useAuth()
  const key = useMemo(
    () => pathTopicChatKey(user?.id, topicId),
    [user?.id, topicId],
  )
  const stream = useCallback(
    (text: string, onDelta: (piece: string) => void) =>
      streamPathTopicChat(topicId, text, onDelta),
    [topicId],
  )

  return useTurn(key, stream)
}

export function useClearPathTopicChat(topicId: string) {
  const { user } = useAuth()
  const client = useQueryClient()

  return useMutation({
    mutationFn: () => clearPathTopicChat(topicId),
    onSuccess: () =>
      client.setQueryData<Chat>(
        pathTopicChatKey(user?.id, topicId),
        (old) => old && { ...old, messages: [] },
      ),
  })
}
