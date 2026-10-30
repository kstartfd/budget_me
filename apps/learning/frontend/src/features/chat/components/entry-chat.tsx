"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { IconMessageCircle, IconTrash } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

import { useChat, useChatTurn, useClearChat } from "../hooks"
import {
  Bubble,
  chatLimits,
  Composer,
  LimitReached,
  useChatFailure,
} from "./chat-ui"

/**
 * A conversation with the user's own AI agent about one entry. It is only
 * offered while an agent that can chat is set up, so an entry of someone
 * without one shows nothing here.
 */
export function EntryChat({
  entryId,
  page = false,
}: {
  entryId: string
  page?: boolean
}) {
  const t = useTranslations("chat")
  const failure = useChatFailure()
  const chat = useChat(entryId)
  const { turn, error, send, sending } = useChatTurn(entryId)
  const clear = useClearChat(entryId)
  const [open, setOpen] = useState(false)
  const [text, setText] = useState("")
  const [confirmClear, setConfirmClear] = useState(false)
  const end = useRef<HTMLDivElement>(null)

  const messages = chat.data?.messages ?? []
  const streaming = turn?.answer.length ?? 0
  const teacher = !!chat.data?.teacher
  const { exhausted, usedPercent } = chatLimits(chat.data)

  useEffect(() => {
    if (open) end.current?.scrollIntoView({ block: "nearest" })
  }, [open, messages.length, streaming, turn?.question])

  if (!chat.data?.available) return null

  const submit = async () => {
    const question = text.trim()
    if (!question || sending) return

    if (await send(question)) setText("")
  }

  return (
    <section
      className={cn(
        page
          ? "bg-card border-border rounded-[14px] border p-3.5"
          : "border-border border-t pt-3.5",
      )}
    >
      <Button
        variant="ghost"
        aria-expanded={open}
        className="text-primary h-auto w-full justify-start gap-2 px-0 py-0 text-sm font-medium"
        onClick={() => setOpen((v) => !v)}
      >
        <IconMessageCircle size={17} />
        {t(open ? "hide" : teacher ? "openTeacher" : "open")}
        {messages.length > 0 && (
          <span className="text-faint text-xs font-normal">
            {t("count", { count: messages.length })}
          </span>
        )}
      </Button>
      {open && (
        <div className="mt-3 space-y-3">
          {messages.length === 0 && !turn && (
            <p className="text-muted-foreground text-[13px] leading-[1.5]">
              {t(teacher ? "hintTeacher" : "hint")}
            </p>
          )}
          <div className="space-y-2" aria-live="polite">
            {messages.map((m) => (
              <Bubble key={m.id} role={m.role}>
                {m.content}
              </Bubble>
            ))}
            {turn && (
              <>
                <Bubble role="user">{turn.question}</Bubble>
                <Bubble role="assistant">{turn.answer || t("typing")}</Bubble>
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
              onSubmit={() => void submit()}
              sending={sending}
              placeholder={t(teacher ? "placeholderTeacher" : "placeholder")}
            />
          )}
          {!exhausted && usedPercent >= 70 && (
            <p className="text-faint text-xs">
              {t("budgetLow", { percent: usedPercent })}
            </p>
          )}
          {messages.length > 0 &&
            (confirmClear ? (
              <div className="flex items-center gap-2">
                <span className="text-sm">{t("clearConfirm")}</span>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={clear.isPending}
                  onClick={() => setConfirmClear(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={clear.isPending}
                  onClick={() =>
                    clear.mutate(undefined, {
                      onSuccess: () => setConfirmClear(false),
                    })
                  }
                >
                  {t("clear")}
                </Button>
              </div>
            ) : (
              <Button
                variant="ghost"
                disabled={sending}
                className="text-muted-foreground h-auto px-0 py-0 text-xs"
                onClick={() => setConfirmClear(true)}
              >
                <IconTrash size={14} />
                {t("clear")}
              </Button>
            ))}
          {clear.isError && (
            <p role="alert" className="text-destructive text-sm">
              {t("clearFailed")}
            </p>
          )}
        </div>
      )}
    </section>
  )
}
