"use client"

import { useTranslations } from "next-intl"
import { IconMessageCircle, IconSparkles } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { useLearning } from "../hooks"
import { LearningModal } from "./learning-modal"

const packs = [
  { kind: "phrases", icon: IconMessageCircle },
  { kind: "words", icon: IconSparkles },
] as const

/**
 * Two ready-made batches the learner's agent writes into the topic: the phrases
 * native speakers really use, and the next most frequent words with a trick to
 * remember each. They land in the inbox like any saved entry.
 */
export function PackModal({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("learning.pack")
  const { generatePack } = useLearning()
  const added = generatePack.data?.length

  return (
    <LearningModal title={t("title")} description={t("hint")} onClose={onClose}>
      <div className="grid gap-2">
        {packs.map(({ kind, icon: Icon }) => (
          <button
            key={kind}
            type="button"
            disabled={generatePack.isPending}
            onClick={() => generatePack.mutate({ topicId, kind })}
            className="border-border bg-background hover:bg-muted/50 flex items-center gap-3 rounded-lg border p-3 text-left transition-colors disabled:opacity-50"
          >
            <span className="bg-muted flex size-9 shrink-0 items-center justify-center rounded-md">
              <Icon size={16} />
            </span>
            <span className="min-w-0">
              <span className="block text-sm font-semibold">
                {t(`${kind}.title`)}
              </span>
              <span className="text-muted-foreground block text-[13px]">
                {t(`${kind}.hint`)}
              </span>
            </span>
          </button>
        ))}
      </div>
      {generatePack.isPending && (
        <p className="text-muted-foreground mt-3 animate-pulse text-[13px]">
          {t("writing")}
        </p>
      )}
      {generatePack.isError && (
        <p role="alert" className="text-destructive mt-3 text-[13px]">
          {generatePack.error.message}
        </p>
      )}
      {added !== undefined && !generatePack.isPending && (
        <div className="mt-3 flex items-center justify-between gap-3">
          <p className="text-[13px]" role="status">
            {t("added", { count: added })}
          </p>
          <Button size="lg" variant="outline" onClick={onClose} className="px-4 text-[13px]">
            {t("close")}
          </Button>
        </div>
      )}
    </LearningModal>
  )
}
