"use client"

import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/features/auth"
import { cn } from "@/lib/utils"
import * as api from "../api"
import { useGradeSet, useSavePlan, useSets } from "../hooks"
import type { Entry, Period, Plan } from "../schemas"
import { FlipCard } from "./flip-card"
import { LearningModal } from "./learning-modal"

type Mode = "day" | "week" | "month"

const perDay = [5, 10, 15, 20, 30]

/**
 * Daily sets: a few new cards every day, a review of the whole week and of the
 * whole month. The learner picks how many cards a day and which side is face up.
 * A day not opened is not lost: its cards join the next set.
 */
export function SetsModal({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("learning.sets")
  const sets = useSets(topicId)
  const [session, setSession] = useState<{ mode: Mode; cards: Entry[] } | null>(null)
  const [starting, setStarting] = useState<Mode | null>(null)
  const [failed, setFailed] = useState(false)

  const start = async (mode: Mode) => {
    if (!sets.data) return

    setFailed(false)

    if (mode === "day") {
      setSession({ mode, cards: sets.data.today })

      return
    }

    setStarting(mode)

    try {
      setSession({ mode, cards: await api.fetchSetPeriod(topicId, mode) })
    } catch {
      setFailed(true)
    } finally {
      setStarting(null)
    }
  }

  return (
    <LearningModal title={t("title")} onClose={onClose}>
      {sets.isPending ? (
        <p className="text-muted-foreground animate-pulse text-sm">{t("loading")}</p>
      ) : sets.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("loadFailed")}
        </p>
      ) : session ? (
        <Sitting
          key={session.mode + session.cards.length}
          topicId={topicId}
          mode={session.mode}
          front={sets.data.plan.front}
          cards={session.cards}
          onBack={() => setSession(null)}
          onNext={(mode) => void start(mode)}
        />
      ) : (
        <Menu
          topicId={topicId}
          data={sets.data}
          starting={starting}
          failed={failed}
          onStart={(mode) => void start(mode)}
        />
      )}
    </LearningModal>
  )
}

function Menu({
  topicId,
  data,
  starting,
  failed,
  onStart,
}: {
  topicId: string
  data: NonNullable<ReturnType<typeof useSets>["data"]>
  starting: Mode | null
  failed: boolean
  onStart: (mode: Mode) => void
}) {
  const t = useTranslations("learning.sets")
  const locale = useLocale()
  const save = useSavePlan(topicId)

  const change = (patch: Partial<Plan>) => save.mutate({ ...data.plan, ...patch })

  const range = (p: Period) => {
    const fmt = (iso: string) =>
      new Date(`${iso}T00:00:00`).toLocaleDateString(locale, { day: "numeric", month: "short" })

    return `${fmt(p.from)} – ${fmt(p.to)}`
  }

  const today = data.today.length
  const rows: { mode: Mode; title: string; hint: string; count: number }[] = [
    {
      mode: "day",
      title: t("today"),
      hint:
        today === 0
          ? t("todayEmpty", { available: data.available })
          : t("todayHint", { fresh: data.fresh, weak: data.weak }),
      count: today,
    },
    {
      mode: "week",
      title: t("week"),
      hint: t("periodHint", { range: range(data.week), pending: data.week.pending, total: data.week.total }),
      count: data.week.pending,
    },
    {
      mode: "month",
      title: t("month"),
      hint: t("periodHint", { range: range(data.month), pending: data.month.pending, total: data.month.total }),
      count: data.month.pending,
    },
  ]

  return (
    <div className="space-y-4">
      <div className="grid gap-2">
        {rows.map(({ mode, title, hint, count }) => (
          <button
            key={mode}
            type="button"
            disabled={count === 0 || starting !== null}
            onClick={() => onStart(mode)}
            className="border-border bg-background hover:bg-muted/50 flex items-center justify-between gap-3 rounded-lg border p-3 text-left transition-colors disabled:opacity-50"
          >
            <span className="min-w-0">
              <span className="block text-sm font-semibold">{title}</span>
              <span className="text-muted-foreground block text-[13px]">{hint}</span>
            </span>
            <span className="bg-secondary flex h-8 min-w-8 shrink-0 items-center justify-center rounded-full px-2 text-[13px] font-semibold tabular-nums">
              {starting === mode ? "…" : count}
            </span>
          </button>
        ))}
      </div>

      {failed && (
        <p role="alert" className="text-destructive text-sm">
          {t("loadFailed")}
        </p>
      )}

      <div className="border-border space-y-3 border-t pt-4">
        <div className="space-y-1.5">
          <p className="text-[13px] font-medium">{t("perDay")}</p>
          <div className="flex flex-wrap gap-2">
            {perDay.map((n) => (
              <Button
                key={n}
                type="button"
                size="sm"
                variant={data.plan.dailyNew === n ? "default" : "outline"}
                aria-pressed={data.plan.dailyNew === n}
                disabled={save.isPending}
                onClick={() => change({ dailyNew: n })}
                className="h-auto rounded-full px-3 py-1 text-xs"
              >
                {n}
              </Button>
            ))}
          </div>
        </div>
        <div className="space-y-1.5">
          <p className="text-[13px] font-medium">{t("front")}</p>
          <div className="flex flex-wrap gap-2">
            {(["translation", "text"] as const).map((f) => (
              <Button
                key={f}
                type="button"
                size="sm"
                variant={data.plan.front === f ? "default" : "outline"}
                aria-pressed={data.plan.front === f}
                disabled={save.isPending}
                onClick={() => change({ front: f })}
                className="h-auto rounded-full px-3 py-1 text-xs"
              >
                {t(`front_${f}`)}
              </Button>
            ))}
          </div>
        </div>
        {save.isError && (
          <p role="alert" className="text-destructive text-xs">
            {t("saveFailed")}
          </p>
        )}
        <p className="text-muted-foreground text-xs leading-relaxed">{t("hint")}</p>
      </div>
    </div>
  )
}

/** One sitting: the cards one at a time, "knew it" or "did not know". */
function Sitting({
  topicId,
  mode,
  front,
  cards,
  onBack,
  onNext,
}: {
  topicId: string
  mode: Mode
  front: "text" | "translation"
  cards: Entry[]
  onBack: () => void
  onNext: (mode: Mode) => void
}) {
  const t = useTranslations("learning.sets")
  const { user } = useAuth()
  const client = useQueryClient()
  const grade = useGradeSet()
  const [queue, setQueue] = useState(cards)
  const [flipped, setFlipped] = useState(false)
  const [known, setKnown] = useState(0)
  const [missed, setMissed] = useState(0)

  const refresh = () =>
    client.invalidateQueries({ queryKey: ["learning", user?.id, "sets", topicId] })

  const card = queue[0]

  if (!card) {
    return (
      <div className="space-y-3 text-center">
        <p className="text-base font-semibold">
          {known + missed === 0 ? t("nothing") : t("finished", { known, total: known + missed })}
        </p>
        <div className="flex flex-wrap justify-center gap-2">
          {mode !== "day" && known + missed > 0 && (
            <Button
              size="lg"
              variant="outline"
              onClick={() => {
                void refresh()
                onNext(mode)
              }}
              className="px-5 text-[13px]"
            >
              {t("nextBatch")}
            </Button>
          )}
          <Button
            size="lg"
            onClick={() => {
              void refresh()
              onBack()
            }}
            className="px-6 text-[13px]"
          >
            {t("done")}
          </Button>
        </div>
      </div>
    )
  }

  const answer = (isKnown: boolean) => {
    if (grade.isPending) return

    grade.mutate(
      { id: card.id, period: mode, known: isKnown },
      {
        onSuccess: () => {
          setFlipped(false)
          // A card not known goes to the end of the sitting and is asked again.
          setQueue((q) => (isKnown ? q.slice(1) : [...q.slice(1), card]))
          if (isKnown) setKnown((n) => n + 1)
          else setMissed((n) => n + 1)
        },
      },
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-muted-foreground text-center text-xs tabular-nums">
        {t("left", { count: queue.length })}
      </p>

      <FlipCard
        key={card.id}
        entry={card}
        front={front}
        flipped={flipped}
        onFlip={() => setFlipped((v) => !v)}
        hint={t("tapToFlip")}
      />

      {grade.isError && (
        <p role="alert" className="text-destructive text-center text-[13px]">
          {t("saveFailed")}
        </p>
      )}

      {!flipped ? (
        <Button size="lg" onClick={() => setFlipped(true)} className="w-full text-sm">
          {t("show")}
        </Button>
      ) : (
        <div className={cn("grid grid-cols-2 gap-2")}>
          <Button
            variant="outline"
            size="lg"
            disabled={grade.isPending}
            onClick={() => answer(false)}
            className="text-[13px]"
          >
            {t("dontKnow")}
          </Button>
          <Button
            size="lg"
            disabled={grade.isPending}
            onClick={() => answer(true)}
            className="text-[13px]"
          >
            {t("know")}
          </Button>
        </div>
      )}
    </div>
  )
}
