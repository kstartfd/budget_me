"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"

import { useExampleChat, useExampleChatTurn } from "../hooks"
import {
  Bubble,
  chatLimits,
  Composer,
  LimitReached,
  useChatFailure,
} from "./chat-ui"

/** Questions worth asking about any sentence; each one sends itself. */
const quick = ["why", "turn", "more"] as const

/**
 * The chat about one sentence of a rule's practice: the teacher has the rule
 * and the sentence in front of them, so the learner can ask why it is built
 * that way, turn it into a question or ask for more like it. The conversation
 * goes when the learner leaves the rule's page, like the sentence itself.
 */
export function ExampleChat({
  ruleId,
  exampleId,
}: {
  ruleId: string
  exampleId: string
}) {
  const t = useTranslations("path.practice.chat")
  const chatT = useTranslations("chat")
  const failure = useChatFailure()
  const chat = useExampleChat(ruleId, exampleId)
  const { turn, error, send, sending } = useExampleChatTurn(ruleId, exampleId)
  const [text, setText] = useState("")
  const end = useRef<HTMLDivElement>(null)

  const messages = chat.data?.messages ?? []
  const streaming = turn?.answer.length ?? 0
  const { exhausted } = chatLimits(chat.data)

  useEffect(() => {
    end.current?.scrollIntoView({ block: "nearest" })
  }, [messages.length, streaming, turn?.question])

  if (chat.isPending) {
    return <p className="text-muted-foreground text-xs">{chatT("typing")}</p>
  }

  if (chat.isError) {
    return (
      <p role="alert" className="text-destructive text-sm">
        {chat.error.message}
      </p>
    )
  }

  if (!chat.data.available) {
    return (
      <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3 text-[13px] leading-relaxed">
        <p className="font-medium">{chatT("needAgent")}</p>
        <p className="text-muted-foreground">{chatT("needAgentHint")}</p>
      </div>
    )
  }

  const ask = async (question: string) => {
    const value = question.trim()
    if (!value || sending) return false

    return send(value)
  }

  return (
    <div className="space-y-3">
      {messages.length === 0 && !turn && (
        <p className="text-muted-foreground text-[13px] leading-[1.5]">
          {t("hint")}
        </p>
      )}
      <div className="flex flex-wrap gap-2">
        {quick.map((key) => (
          <Button
            key={key}
            variant="outline"
            size="sm"
            disabled={sending || exhausted}
            onClick={() => void ask(t(`quick.${key}.prompt`))}
            className="h-auto rounded-full px-3 py-1 text-xs"
          >
            {t(`quick.${key}.label`)}
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
            <Bubble role="assistant">{turn.answer || chatT("typing")}</Bubble>
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
    </div>
  )
}
