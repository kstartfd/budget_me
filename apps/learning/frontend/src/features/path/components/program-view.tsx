"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronRight, IconChevronUp } from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { Path } from "../schemas"
import { TopicRunner } from "./topic-runner"

const statusBadge: Record<
  string,
  { variant: "default" | "outline"; className?: string }
> = {
  not_started: { variant: "outline" },
  in_progress: { variant: "default" },
  completed: {
    variant: "outline",
    className: "bg-success/10 text-success border-transparent",
  },
}

const badgeShape = "h-auto rounded-full px-2.5 py-[3px] text-xs"

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
        {path.topics.map((topic, index) => {
          const open = openTopic === topic.id
          const newUnit =
            !!topic.unit && topic.unit !== path.topics[index - 1]?.unit

          return (
            <div key={topic.id} className="space-y-3">
              {newUnit && (
                <h2 className="text-muted-foreground pt-2 text-xs font-semibold tracking-wider uppercase">
                  {topic.unit}
                </h2>
              )}
            <div
              className={cn(
                "bg-background min-w-0 overflow-hidden rounded-lg border",
                topic.status === "in_progress"
                  ? "border-foreground"
                  : "border-border",
              )}
            >
              <button
                type="button"
                aria-expanded={open}
                onClick={() => setOpenTopic(open ? null : topic.id)}
                className={cn(
                  "hover:bg-muted/50 flex w-full min-w-0 cursor-pointer items-center gap-3 px-4 py-3.5 text-left transition-colors",
                  topic.status === "in_progress" && "bg-muted",
                  open && "border-border border-b",
                )}
              >
                <span className="bg-secondary flex size-8 shrink-0 items-center justify-center rounded-full text-[13px] font-semibold">
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
                  variant={statusBadge[topic.status].variant}
                  className={cn(
                    badgeShape,
                    "font-medium",
                    statusBadge[topic.status].className,
                  )}
                >
                  {t(`topicStatus.${topic.status}`)}
                </Badge>
                {open ? (
                  <IconChevronUp size={13} className="text-faint shrink-0" />
                ) : (
                  <IconChevronRight size={13} className="text-faint shrink-0" />
                )}
              </button>
              {open && <TopicRunner topicId={topic.id} />}
            </div>
            </div>
          )
        })}
      </section>
    </div>
  )
}
