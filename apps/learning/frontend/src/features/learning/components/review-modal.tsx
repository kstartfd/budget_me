"use client"

import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/features/auth"
import { useReview, useReviewEntry } from "../hooks"
import type { Entry, Grade } from "../schemas"
import { EntryAudioButton } from "./audio-button"
import { LearningModal } from "./learning-modal"

/**
 * One sitting of spaced repetition: the cards due now, one at a time. The front
 * shows the word or phrase; the learner tries to recall it, reveals the card and
 * says how it went. A forgotten card comes back at the end of the sitting, and the
 * server moves every card along its schedule.
 */
export function ReviewModal({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("learning.review")
  const review = useReview(topicId)

  return (
    <LearningModal title={t("title")} onClose={onClose}>
      {review.isPending ? (
        <p className="text-muted-foreground animate-pulse text-sm">{t("loading")}</p>
      ) : review.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("loadFailed")}
        </p>
      ) : (
        <Sitting topicId={topicId} cards={review.data.entries} onClose={onClose} />
      )}
    </LearningModal>
  )
}

function Sitting({
  topicId,
  cards,
  onClose,
}: {
  topicId: string
  cards: Entry[]
  onClose: () => void
}) {
  const t = useTranslations("learning.review")
  const { user } = useAuth()
  const client = useQueryClient()
  const grade = useReviewEntry()
  const [queue, setQueue] = useState(cards)
  const [shown, setShown] = useState(false)
  const [done, setDone] = useState(0)
  const [again, setAgain] = useState(0)

  const finish = async () => {
    await client.invalidateQueries({ queryKey: ["learning", user?.id, "review", topicId] })
    await client.invalidateQueries({ queryKey: ["learning", user?.id, "entries", topicId] })
    onClose()
  }

  const card = queue[0]

  if (!card) {
    return (
      <div className="space-y-3 text-center">
        <p className="text-base font-semibold">
          {done === 0 ? t("nothing") : t("finished", { count: done })}
        </p>
        {again > 0 && (
          <p className="text-muted-foreground text-[13px]">{t("againCount", { count: again })}</p>
        )}
        <Button size="lg" onClick={() => void finish()} className="px-6 text-[13px]">
          {t("close")}
        </Button>
      </div>
    )
  }

  const answer = (g: Grade) => {
    if (grade.isPending) return

    grade.mutate(
      { id: card.id, grade: g },
      {
        onSuccess: () => {
          setShown(false)
          setQueue((q) => (g === "again" ? [...q.slice(1), card] : q.slice(1)))

          if (g === "again") setAgain((n) => n + 1)
          else setDone((n) => n + 1)
        },
      },
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-muted-foreground text-center text-xs tabular-nums">
        {t("left", { count: queue.length })}
      </p>

      <div className="border-border bg-background space-y-3 rounded-lg border p-5 text-center">
        <p className="text-xl font-semibold break-words">{card.text}</p>
        {card.transcription && (
          <p className="text-muted-foreground font-mono text-sm">{card.transcription}</p>
        )}
        <div className="flex justify-center">
          <EntryAudioButton entryId={card.id} />
        </div>

        {shown && (
          <div className="border-border space-y-2 border-t pt-3 text-left text-sm">
            {card.translation && <p className="font-semibold">{card.translation}</p>}
            {card.explanation && (
              <p className="text-muted-foreground leading-relaxed">{card.explanation}</p>
            )}
            {card.example && (
              <p className="leading-relaxed italic">{card.example}</p>
            )}
          </div>
        )}
      </div>

      {grade.isError && (
        <p role="alert" className="text-destructive text-center text-[13px]">
          {t("saveFailed")}
        </p>
      )}

      {!shown ? (
        <Button size="lg" onClick={() => setShown(true)} className="w-full text-sm">
          {t("show")}
        </Button>
      ) : (
        <div className="grid grid-cols-3 gap-2">
          <Button
            variant="outline"
            size="lg"
            disabled={grade.isPending}
            onClick={() => answer("again")}
            className="text-[13px]"
          >
            {t("again")}
          </Button>
          <Button
            size="lg"
            disabled={grade.isPending}
            onClick={() => answer("good")}
            className="text-[13px]"
          >
            {t("good")}
          </Button>
          <Button
            variant="outline"
            size="lg"
            disabled={grade.isPending}
            onClick={() => answer("easy")}
            className="text-[13px]"
          >
            {t("easy")}
          </Button>
        </div>
      )}
    </div>
  )
}
