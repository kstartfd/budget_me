"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { IconMessageCircle, IconSend2, IconTrash } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

import { ChatError, chatErrorCode } from "../api"
import { useChat, useChatTurn, useClearChat } from "../hooks"
import { maxQuestionLength } from "../schemas"

function Bubble({
  role,
  children,
}: {
  role: "user" | "assistant"
  children: React.ReactNode
}) {
  const mine = role === "user"

  return (
    <div className={cn("flex", mine ? "justify-end" : "justify-start")}>
      <p
        className={cn(
          "max-w-[88%] rounded-2xl px-3.5 py-2 text-[13px] leading-[1.5] break-words whitespace-pre-wrap",
          mine
            ? "bg-primary text-primary-foreground rounded-br-md"
            : "bg-secondary text-foreground rounded-bl-md",
        )}
      >
        {children}
      </p>
    </div>
  )
}

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
  const chat = useChat(entryId)
  const { turn, error, send, sending } = useChatTurn(entryId)
  const clear = useClearChat(entryId)
  const [open, setOpen] = useState(false)
  const [text, setText] = useState("")
  const [confirmClear, setConfirmClear] = useState(false)
  const end = useRef<HTMLDivElement>(null)

  const messages = chat.data?.messages ?? []
  const streaming = turn?.answer.length ?? 0

  useEffect(() => {
    if (open) end.current?.scrollIntoView({ block: "nearest" })
  }, [open, messages.length, streaming, turn?.question])

  if (!chat.data?.available) return null

  const submit = async () => {
    const question = text.trim()
    if (!question || sending) return

    if (await send(question)) setText("")
  }

  const failure = (e: ChatError) => {
    switch (e.code) {
      case chatErrorCode.unavailable:
        return t("unavailable")
      case chatErrorCode.busy:
        return t("busy")
      // The backend words these for the learner (no agent, key rejected, ...).
      case chatErrorCode.invalid:
        return e.message
      default:
        return t("failed")
    }
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
        {t(open ? "hide" : "open")}
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
              {t("hint")}
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
          <div className="flex items-end gap-2">
            <Textarea
              value={text}
              maxLength={maxQuestionLength}
              rows={1}
              disabled={sending}
              placeholder={t("placeholder")}
              aria-label={t("placeholder")}
              className="max-h-40 min-h-10 flex-1 resize-none rounded-xl px-3 py-2.5 text-[13px] md:text-[13px]"
              onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => {
                if (
                  e.key === "Enter" &&
                  !e.shiftKey &&
                  !e.nativeEvent.isComposing
                ) {
                  e.preventDefault()
                  void submit()
                }
              }}
            />
            <Button
              size="icon"
              aria-label={t("send")}
              disabled={sending || text.trim() === ""}
              className="size-10 shrink-0 rounded-xl"
              onClick={() => void submit()}
            >
              <IconSend2 size={18} />
            </Button>
          </div>
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
