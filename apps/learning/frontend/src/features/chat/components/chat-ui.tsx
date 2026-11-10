"use client"

import { useFormatter, useTranslations } from "next-intl"
import { IconArrowUp, IconSchool } from "@tabler/icons-react"

import { InlineMarkdown } from "@/components/inline-markdown"
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
    <div
      className={cn("flex gap-2.5", mine ? "justify-end" : "justify-start")}
    >
      {!mine && (
        <span
          aria-hidden
          className="bg-primary text-primary-foreground flex size-7 shrink-0 items-center justify-center rounded-full"
        >
          <IconSchool size={13} />
        </span>
      )}
      <p
        className={cn(
          "max-w-[88%] rounded-lg px-4 py-3 text-sm leading-relaxed break-words whitespace-pre-wrap lg:max-w-[640px]",
          mine
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-foreground",
        )}
      >
        {!mine && typeof children === "string" ? (
          <InlineMarkdown text={children} />
        ) : (
          children
        )}
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
    <p className="text-muted-foreground bg-muted rounded-lg px-4 py-3 text-sm leading-relaxed">
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
    <div className="border-input bg-background focus-within:border-ring flex items-end gap-2 rounded-lg border py-2 pr-2 pl-4 transition-colors">
      <Textarea
        value={value}
        maxLength={maxQuestionLength}
        rows={1}
        disabled={sending}
        placeholder={placeholder}
        aria-label={placeholder}
        className="max-h-40 min-h-9 flex-1 resize-none border-0 bg-transparent px-0 py-2 text-sm shadow-none focus-visible:ring-0 disabled:bg-transparent md:text-sm dark:bg-transparent dark:disabled:bg-transparent"
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
            e.preventDefault()
            onSubmit()
          }
        }}
      />
      <Button
        size="icon-lg"
        aria-label={t("send")}
        disabled={sending || value.trim() === ""}
        className="shrink-0"
        onClick={onSubmit}
      >
        <IconArrowUp size={16} />
      </Button>
    </div>
  )
}
