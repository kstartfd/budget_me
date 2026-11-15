"use client";

import { useTranslations } from "next-intl";
import {
  IconLoader2,
  IconMessageCircle,
  IconSparkles,
} from "@tabler/icons-react";
import { useLearning } from "../hooks";
import { LearningModal } from "./learning-modal";

const packs = [
  { kind: "phrases", icon: IconMessageCircle },
  { kind: "words", icon: IconSparkles },
] as const;

/**
 * Two ready-made batches the learner's agent writes into the topic: the phrases
 * native speakers really use, and the next most frequent words with a trick to
 * remember each. They land in the inbox like any saved entry.
 */
export function PackModal({
  topicId,
  onClose,
}: {
  topicId: string;
  onClose: () => void;
}) {
  const t = useTranslations("learning.pack");
  const { generatePack } = useLearning();

  return (
    <LearningModal title={t("title")} description={t("hint")} onClose={onClose}>
      {generatePack.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="flex flex-col items-center gap-3 py-8 text-center"
        >
          <IconLoader2
            size={32}
            className="text-muted-foreground animate-spin"
          />
          <p className="text-sm font-medium">{t("writing")}</p>
        </div>
      ) : (
        <div className="grid gap-2">
          {packs.map(({ kind, icon: Icon }) => (
            <button
              key={kind}
              type="button"
              onClick={() =>
                generatePack.mutate({ topicId, kind }, { onSuccess: onClose })
              }
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
      )}
      {generatePack.isError && (
        <p role="alert" className="text-destructive mt-3 text-[13px]">
          {generatePack.error.message}
        </p>
      )}
    </LearningModal>
  );
}
