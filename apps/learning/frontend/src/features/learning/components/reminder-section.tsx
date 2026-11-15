"use client"

import { useTranslations } from "next-intl"

import { SectionKicker, sectionCard, sectionHint } from "@/components/section"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { useAuth } from "@/features/auth"
import { cn } from "@/lib/utils"

import { useReminder, useSaveReminder } from "../hooks"

/**
 * The daily nudge from the bot when cards are due for review. It is off until the
 * learner switches it on, goes out in their own morning (the time zone is read
 * from this browser when they switch it on) and needs a Telegram sign-in, since
 * the bot can only write to someone who has opened it.
 */
export function ReminderSection() {
  const t = useTranslations("learning.reminder")
  const { user } = useAuth()
  const reminder = useReminder()
  const save = useSaveReminder()

  if (!user?.telegram_id) return null

  const enabled = reminder.data?.enabled ?? false

  const toggle = (on: boolean) => {
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
    save.mutate({ enabled: on, timezone })
  }

  return (
    <section className="flex flex-col gap-3">
      <SectionKicker>{t("kicker")}</SectionKicker>

      {reminder.isPending ? (
        <Skeleton className="h-[72px] rounded-[12px]" />
      ) : (
        <div className={cn(sectionCard, "flex items-center justify-between gap-4 p-4")}>
          <div className="min-w-0">
            <p className="text-sm font-medium">{t("title")}</p>
            <p className={sectionHint}>{t("hint")}</p>
          </div>
          <Switch
            checked={enabled}
            disabled={save.isPending}
            aria-label={t("title")}
            onCheckedChange={toggle}
          />
        </div>
      )}

      {(reminder.isError || save.isError) && (
        <p role="alert" className="text-destructive px-0.5 text-xs">
          {t("failed")}
        </p>
      )}
    </section>
  )
}
