"use client"

import { useFormatter, useTranslations } from "next-intl"
import { IconSend2 } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

import { ChatError, chatErrorCode } from "../api"
import { maxQuestionLength, type Chat } from "../schemas"

export function Bubble({
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

/** When a window ends, as a time of day; empty if it is not known. */
function useResetTime() {
  const format = useFormatter()

  return (iso?: string | null) =>
    iso
      ? format.dateTime(new Date(iso), { hour: "numeric", minute: "numeric" })
      : ""
}

/** The words for a refused or failed turn. */
export function useChatFailure() {
  const t = useTranslations("chat")
  const resetTime = useResetTime()

  return (e: ChatError) => {
    switch (e.code) {
      case chatErrorCode.unavailable:
        return t("unavailable")
      case chatErrorCode.busy:
        return t("busy")
      case chatErrorCode.tooMany:
        return t("tooMany")
      case chatErrorCode.budget:
        return t("budgetReached", { time: resetTime(e.details[0]) })
      case chatErrorCode.daily:
        return t("dailyReached", { time: resetTime(e.details[0]) })
      // The backend words these for the learner (no agent, key rejected, ...).
      case chatErrorCode.invalid:
        return e.message
      default:
        return t("failed")
    }
  }
}

/** How much of the chat's and the day's tokens are used. */
export function chatLimits(chat: Chat | undefined) {
  const budget = chat?.budget
  const daily = chat?.daily
  const dailyExhausted = !!daily && daily.used >= daily.limit
  const exhausted = (!!budget && budget.used >= budget.limit) || dailyExhausted
  const usedPercent = budget
    ? Math.round((budget.used / budget.limit) * 100)
    : 0

  return { budget, daily, dailyExhausted, exhausted, usedPercent }
}

/** Replaces the composer once the chat or the day has no tokens left. */
export function LimitReached({ chat }: { chat: Chat | undefined }) {
  const t = useTranslations("chat")
  const resetTime = useResetTime()
  const { budget, daily, dailyExhausted } = chatLimits(chat)

  return (
    <p className="text-muted-foreground bg-secondary rounded-xl px-3.5 py-2.5 text-[13px] leading-[1.5]">
      {dailyExhausted
        ? t("dailyReached", { time: resetTime(daily?.resetsAt) })
        : t("budgetReached", { time: resetTime(budget?.resetsAt) })}
    </p>
  )
}

export function Composer({
  value,
  onChange,
  onSubmit,
  sending,
  placeholder,
}: {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  sending: boolean
  placeholder: string
}) {
  const t = useTranslations("chat")

  return (
    <div className="flex items-end gap-2">
      <Textarea
        value={value}
        maxLength={maxQuestionLength}
        rows={1}
        disabled={sending}
        placeholder={placeholder}
        aria-label={placeholder}
        className="max-h-40 min-h-10 flex-1 resize-none rounded-xl px-3 py-2.5 text-[13px] md:text-[13px]"
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
            e.preventDefault()
            onSubmit()
          }
        }}
      />
      <Button
        size="icon"
        aria-label={t("send")}
        disabled={sending || value.trim() === ""}
        className="size-10 shrink-0 rounded-xl"
        onClick={onSubmit}
      >
        <IconSend2 size={18} />
      </Button>
    </div>
  )
}
