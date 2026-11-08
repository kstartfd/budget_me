"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconChevronRight } from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { Path } from "../schemas"
import { TopicRunner } from "./topic-runner"

const statusVariant: Record<string, "secondary" | "default" | "outline"> = {
  not_started: "secondary",
  in_progress: "default",
  completed: "outline",
}

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
    <div className="space-y-[18px]">
      <section className="bg-card border-border space-y-3 rounded-[22px] border p-5 shadow-[var(--shadow-card)]">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-bold">
            {t("programTitle", { level: path.level })}
          </h2>
          <Badge variant="secondary">
            {t("progressTopics", { done, total: path.topics.length })}
          </Badge>
        </div>
        {path.summary && (
          <p className="text-muted-foreground text-sm leading-relaxed">
            {path.summary}
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={onStartFinal}
            className="rounded-[16px] px-4 py-2.5 text-sm font-semibold"
          >
            {t("finalTest")}
          </Button>
          {confirmReset ? (
            <>
              <Button
                variant="destructive"
                onClick={() => {
                  setConfirmReset(false)
                  onReset()
                }}
                className="rounded-[16px] px-4 py-2.5 text-sm font-semibold"
              >
                {t("resetConfirm")}
              </Button>
              <Button
                variant="ghost"
                onClick={() => setConfirmReset(false)}
                className="rounded-[16px] px-4 py-2.5 text-sm font-semibold"
              >
                {t("cancel")}
              </Button>
            </>
          ) : (
            <Button
              variant="ghost"
              onClick={() => setConfirmReset(true)}
              className="rounded-[16px] px-4 py-2.5 text-sm font-semibold"
            >
              {t("reset")}
            </Button>
          )}
        </div>
      </section>

      <section className="space-y-2.5">
        {path.topics.map((topic) => (
          <div key={topic.id} className="space-y-2.5">
            <button
              type="button"
              onClick={() =>
                setOpenTopic(openTopic === topic.id ? null : topic.id)
              }
              className="bg-card border-border hover:border-primary/30 flex min-w-0 w-full items-center gap-3.5 rounded-[18px] border p-4 text-left shadow-[var(--shadow-card)] transition-colors"
            >
              <span className="bg-[color:var(--primary-wash)] text-primary flex size-11 shrink-0 items-center justify-center rounded-full text-sm font-bold">
                {topic.position + 1}
              </span>
              <div className="min-w-0 flex-1">
                <h3 className="line-clamp-2 text-base leading-snug font-semibold break-words">
                  {topic.title}
                </h3>
                <p className="text-muted-foreground mt-0.5 text-xs break-words">
                  {t(`skill.${topic.skill}`)}
                </p>
              </div>
              <Badge variant={statusVariant[topic.status]}>
                {t(`topicStatus.${topic.status}`)}
              </Badge>
              <IconChevronRight
                size={17}
                className={`text-faint shrink-0 transition-transform ${openTopic === topic.id ? "rotate-90" : ""}`}
              />
            </button>
            {openTopic === topic.id && <TopicRunner topicId={topic.id} />}
          </div>
        ))}
      </section>
    </div>
  )
}
