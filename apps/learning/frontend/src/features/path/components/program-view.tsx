"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronDown, IconChevronRight } from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { Path } from "../schemas"
import { TopicRunner } from "./topic-runner"

const statusVariant: Record<string, "secondary" | "default" | "outline"> = {
  not_started: "outline",
  in_progress: "default",
  completed: "outline",
}

const badgeShape = "h-auto rounded-md px-2.5 py-[3px] text-xs"

/** ProgramView is the active program: level, summary, topics, actions. */
export function ProgramView({
  path,
  onStartFinal,
  onReset,
}: {
  path: Path
  onStartFinal: () => void
  onReset: () => void
}) {
  const t = useTranslations("path")
  const [openTopic, setOpenTopic] = useState<string | null>(null)
  const [confirmReset, setConfirmReset] = useState(false)
  const done = path.topics.filter((topic) => topic.status === "completed").length

  return (
    <div className="space-y-4">
      <section className="bg-background border-border space-y-3 rounded-lg border p-5">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-base font-semibold">
            {t("programTitle", { level: path.level })}
          </h2>
          <Badge variant="secondary" className={cn(badgeShape, "font-semibold")}>
            {t("progressTopics", { done, total: path.topics.length })}
          </Badge>
        </div>
        {path.summary && (
          <p className="text-muted-foreground text-sm leading-relaxed">
            {path.summary}
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          <Button size="lg" onClick={onStartFinal} className="px-4 text-[13px]">
            {t("finalTest")}
          </Button>
          {confirmReset ? (
            <>
              <Button
                variant="destructive"
                size="lg"
                onClick={() => {
                  setConfirmReset(false)
                  onReset()
                }}
                className="px-4 text-[13px]"
              >
                {t("resetConfirm")}
              </Button>
              <Button
                variant="ghost"
                size="lg"
                onClick={() => setConfirmReset(false)}
                className="px-4 text-[13px]"
              >
                {t("cancel")}
              </Button>
            </>
          ) : (
            <Button
              variant="outline"
              size="lg"
              onClick={() => setConfirmReset(true)}
              className="px-4 text-[13px]"
            >
              {t("reset")}
            </Button>
          )}
        </div>
      </section>

      <section className="space-y-3">
        {path.topics.map((topic) => {
          const open = openTopic === topic.id

          return (
            <div
              key={topic.id}
              className="bg-background border-border min-w-0 overflow-hidden rounded-lg border"
            >
              <button
                type="button"
                aria-expanded={open}
                onClick={() => setOpenTopic(open ? null : topic.id)}
                className={cn(
                  "hover:bg-muted/50 flex w-full min-w-0 cursor-pointer items-center gap-3 px-4 py-3.5 text-left transition-colors",
                  open && "border-border border-b",
                )}
              >
                <span className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-full text-[13px] font-semibold">
                  {topic.position + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <h3 className="line-clamp-2 text-sm leading-snug font-semibold break-words">
                    {topic.title}
                  </h3>
                  <p className="text-faint mt-0.5 text-xs break-words">
                    {t(`skill.${topic.skill}`)}
                  </p>
                </div>
                <Badge
                  variant={statusVariant[topic.status]}
                  className={cn(badgeShape, "font-medium")}
                >
                  {t(`topicStatus.${topic.status}`)}
                </Badge>
                {open ? (
                  <IconChevronDown size={13} className="text-faint shrink-0" />
                ) : (
                  <IconChevronRight size={13} className="text-faint shrink-0" />
                )}
              </button>
              {open && <TopicRunner topicId={topic.id} />}
            </div>
          )
        })}
      </section>
    </div>
  )
}
