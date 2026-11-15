"use client"

import { useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import { useAddEntries, useLearning } from "../hooks"
import { useAiSettings } from "@/features/ai"
import { entryInputSchema, type Entry, type EntryInput } from "../schemas"
import { LearningModal } from "./learning-modal"

function EditEntryForm({
  entry,
  topicId,
  kind,
  onClose,
}: {
  entry?: Entry
  topicId: string
  kind: EntryInput["kind"]
  onClose: () => void
}) {
  const t = useTranslations("learning")
  const { saveEntry } = useLearning()
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<EntryInput>({
    resolver: zodResolver(entryInputSchema),
    defaultValues: entry ?? {
      topicId,
      kind,
      text: "",
      transcription: "",
      translation: "",
      explanation: "",
      nativeUsage: "",
      example: "",
      rule: "",
      source: "",
      status: "inbox",
    },
  })
  return (
    <LearningModal
      title={t(entry ? "editEntry" : "newEntry")}
      description={t("entryHint")}
      onClose={onClose}
    >
      <form
        className="space-y-4"
        onSubmit={handleSubmit((values) =>
          saveEntry.mutate(
            { ...values, id: entry?.id },
            { onSuccess: onClose },
          ),
        )}
      >
        <Controller
          control={control}
          name="kind"
          render={({ field }) => (
            <div
              className="bg-muted flex h-9 gap-0.5 rounded-md p-1"
              role="group"
              aria-label={t("kind")}
            >
              {(["word", "phrase"] as const).map((k) => (
                <Button
                  key={k}
                  type="button"
                  variant="ghost"
                  aria-pressed={field.value === k}
                  onClick={() => field.onChange(k)}
                  className={cn(
                    "h-full flex-1 rounded-sm text-[13px] font-medium",
                    field.value === k
                      ? "bg-background text-foreground hover:bg-background"
                      : "text-muted-foreground",
                  )}
                >
                  {t(k)}
                </Button>
              ))}
            </div>
          )}
        />
        <label className="block space-y-2 text-sm font-medium">
          <span>{t("text")}</span>
          <Textarea
            autoFocus
            className="bg-background min-h-20 text-sm md:text-sm"
            maxLength={1000}
            placeholder={t("textPlaceholder")}
            aria-invalid={!!errors.text}
            {...register("text")}
          />
        </label>
        <label className="block space-y-2 text-sm font-medium">
          <span>{t("transcription")}</span>
          <Input
            className="bg-background h-9 text-sm md:text-sm"
            maxLength={4000}
            placeholder="/hɪt ðə rəʊd/"
            {...register("transcription")}
          />
        </label>
        {(["translation", "explanation", "rule", "nativeUsage", "example"] as const).map(
          (name) => (
            <label key={name} className="block space-y-2 text-sm font-medium">
              <span>{t(name)}</span>
              <Textarea
                className="bg-background min-h-16 text-sm md:text-sm"
                maxLength={4000}
                placeholder={t(`${name}Placeholder`)}
                {...register(name)}
              />
            </label>
          ),
        )}
        <label className="block space-y-2 text-sm font-medium">
          <span>{t("source")}</span>
          <Input
            className="bg-background h-9 text-sm md:text-sm"
            maxLength={1000}
            placeholder={t("sourcePlaceholder")}
            {...register("source")}
          />
        </label>
        <Controller
          control={control}
          name="status"
          render={({ field }) => (
            <div
              className="flex flex-wrap gap-2"
              role="group"
              aria-label={t("status")}
            >
              {(["inbox", "learning", "known"] as const).map((status) => (
                <Button
                  key={status}
                  type="button"
                  variant="outline"
                  aria-pressed={field.value === status}
                  onClick={() => field.onChange(status)}
                  className={cn(
                    "h-7 px-3 text-xs font-medium",
                    field.value === status &&
                      "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground border-transparent",
                  )}
                >
                  {t(status)}
                </Button>
              ))}
            </div>
          )}
        />
        {Object.keys(errors).length > 0 && (
          <p role="alert" className="text-destructive text-sm">
            {t("checkFields")}
          </p>
        )}
        {saveEntry.isError && (
          <p role="alert" className="text-destructive text-sm">
            {t("saveError")}
          </p>
        )}
        <Button
          type="submit"
          disabled={saveEntry.isPending}
          className="h-10 w-full text-sm"
        >
          {t(saveEntry.isPending ? "saving" : "save")}
        </Button>
      </form>
    </LearningModal>
  )
}

/**
 * A new entry is just the phrase: one line. It is saved at once and, when an AI
 * agent is connected, the agent writes the transcription, translation,
 * breakdown, rule, usage and example in the background. One word is a word,
 * anything longer a phrase. The full card is for editing an entry afterwards.
 */
export function EntryForm(props: {
  entry?: Entry
  topicId: string
  kind: EntryInput["kind"]
  onClose: () => void
}) {
  return props.entry ? (
    <EditEntryForm {...props} entry={props.entry} />
  ) : (
    <QuickAdd topicId={props.topicId} onClose={props.onClose} />
  )
}

function QuickAdd({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("learning")
  const q = useTranslations("learning.quick")
  const ai = useAiSettings()
  const add = useAddEntries()
  const [text, setText] = useState("")

  const value = text.trim()

  return (
    <LearningModal title={t("newEntry")} onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          if (!value) return

          add.mutate(
            [{ topicId, kind: /\s/.test(value) ? "phrase" : "word", text: value }],
            { onSuccess: onClose },
          )
        }}
      >
        <Input
          autoFocus
          value={text}
          maxLength={1000}
          onChange={(e) => setText(e.target.value)}
          placeholder={q("placeholder")}
          aria-label={q("label")}
          className="bg-background h-11 text-base md:text-base"
        />

        <p className="text-muted-foreground text-[13px] leading-relaxed">
          {ai.data ? q("withAgent") : q("withoutAgent")}
        </p>

        {add.isError && (
          <p role="alert" className="text-destructive text-sm">
            {t("saveError")}
          </p>
        )}

        <Button
          type="submit"
          disabled={add.isPending || !value}
          className="h-10 w-full text-sm"
        >
          {add.isPending ? t("saving") : t("save")}
        </Button>
      </form>
    </LearningModal>
  )
}
