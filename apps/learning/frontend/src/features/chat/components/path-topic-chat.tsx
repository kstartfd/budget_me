"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconMessageCircle, IconTrash } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"

import {
  useClearPathTopicChat,
  usePathTopicChat,
  usePathTopicChatTurn,
} from "../hooks"
import {
  Bubble,
  chatLimits,
  Composer,
  LimitReached,
  useChatFailure,
} from "./chat-ui"

/** Ways to use the teacher on a topic; each one sends a ready question. */
const modes = ["dialogue", "natural", "pronounce", "cards", "quick"] as const

/**
 * The chat of one topic of the program: the teacher knows the topic and its
 * rule, so the learner can ask, rehearse a dialogue or collect words on it.
 * The link leads on to the general chat.
 */
export function PathTopicChat({
  topicId,
  topicTitle,
  generalHref,
}: {
  topicId: string
  topicTitle: string
  generalHref: string
}) {
  const t = useTranslations("path.chat")
  const chatT = useTranslations("chat")
  const failure = useChatFailure()
  const chat = usePathTopicChat(topicId)
  const { turn, error, send, sending } = usePathTopicChatTurn(topicId)
  const clear = useClearPathTopicChat(topicId)
  const [open, setOpen] = useState(false)
  const [text, setText] = useState("")
  const end = useRef<HTMLDivElement>(null)

  const messages = chat.data?.messages ?? []
  const streaming = turn?.answer.length ?? 0
  const { exhausted } = chatLimits(chat.data)

  useEffect(() => {
    if (open) end.current?.scrollIntoView({ block: "nearest" })
  }, [open, messages.length, streaming, turn?.question])

  if (!chat.data?.available) return null

  const ask = async (question: string) => {
    const value = question.trim()
    if (!value || sending) return false

    return send(value)
  }

  return (
    <section className="border-border space-y-3 border-t pt-4">
      <div className="flex items-center justify-between gap-3">
        <Button
          variant="ghost"
          aria-expanded={open}
          className="text-primary h-auto justify-start gap-2 px-0 py-0 text-sm font-medium"
          onClick={() => setOpen((v) => !v)}
        >
          <IconMessageCircle size={17} />
          {t(open ? "hide" : "open")}
          {messages.length > 0 && (
            <span className="text-faint text-xs font-normal">
              {chatT("count", { count: messages.length })}
            </span>
          )}
        </Button>
        <Link
          href={generalHref}
          className="text-muted-foreground hover:text-foreground text-xs underline-offset-2 hover:underline"
        >
          {t("general")}
        </Link>
      </div>

      {open && (
        <div className="space-y-3">
          {messages.length === 0 && !turn && (
            <p className="text-muted-foreground text-[13px] leading-[1.5]">
              {t("hint")}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            {modes.map((mode) => (
              <Button
                key={mode}
                variant="outline"
                size="sm"
                disabled={sending || exhausted}
                onClick={() =>
                  void ask(t(`mode.${mode}.prompt`, { topic: topicTitle }))
                }
                className="h-auto rounded-full px-3 py-1 text-xs"
              >
                {t(`mode.${mode}.label`)}
              </Button>
            ))}
          </div>
          <div className="space-y-2" aria-live="polite">
            {messages.map((m) => (
              <Bubble key={m.id} role={m.role}>
                {m.content}
              </Bubble>
            ))}
            {turn && (
              <>
                <Bubble role="user">{turn.question}</Bubble>
                <Bubble role="assistant">
                  {turn.answer || chatT("typing")}
                </Bubble>
              </>
            )}
            <div ref={end} />
          </div>
          {error && (
            <p role="alert" className="text-destructive text-sm">
              {failure(error)}
            </p>
          )}
          {exhausted ? (
            <LimitReached chat={chat.data} />
          ) : (
            <Composer
              value={text}
              onChange={setText}
              onSubmit={() => void ask(text).then((ok) => ok && setText(""))}
              sending={sending}
              placeholder={chatT("placeholderTeacher")}
            />
          )}
          {messages.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              disabled={clear.isPending || sending}
              onClick={() => clear.mutate()}
              className="text-muted-foreground h-auto gap-1.5 px-0 py-0 text-xs"
            >
              <IconTrash size={13} />
              {chatT("clear")}
            </Button>
          )}
        </div>
      )}
    </section>
  )
}
