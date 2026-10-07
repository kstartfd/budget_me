"use client"

import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import { useLearning } from "../hooks"
import { entryInputSchema, type Entry, type EntryInput } from "../schemas"
import { LearningModal } from "./learning-modal"

export function EntryForm({
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
              className="bg-secondary flex gap-1 rounded-2xl p-1"
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
                    "h-10 flex-1 rounded-xl",
                    field.value === k && "bg-card text-primary shadow-sm",
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
            className="bg-secondary min-h-20 rounded-2xl text-base md:text-base"
            maxLength={1000}
            placeholder={t("textPlaceholder")}
            aria-invalid={!!errors.text}
            {...register("text")}
          />
        </label>
        <label className="block space-y-2 text-sm font-medium">
          <span>{t("transcription")}</span>
          <Input
            className="bg-secondary h-11 rounded-xl"
            maxLength={4000}
            placeholder="/hɪt ðə rəʊd/"
            {...register("transcription")}
          />
        </label>
        {(["translation", "explanation", "rule", "nativeUsage", "example", "source"] as const).map(
          (name) => (
            <label key={name} className="block space-y-2 text-sm font-medium">
              <span>{t(name)}</span>
              <Textarea
                className="bg-secondary min-h-16 rounded-xl text-sm md:text-sm"
                maxLength={4000}
                placeholder={t(`${name}Placeholder`)}
                {...register(name)}
              />
            </label>
          ),
        )}
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
                  variant="secondary"
                  aria-pressed={field.value === status}
                  onClick={() => field.onChange(status)}
                  className={cn(
                    "rounded-full text-xs",
                    field.value === status &&
                      "bg-primary/15 text-primary ring-primary/30 ring-1",
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
          className="h-12 w-full rounded-2xl font-bold"
        >
          {t(saveEntry.isPending ? "saving" : "save")}
        </Button>
      </form>
    </LearningModal>
  )
}
