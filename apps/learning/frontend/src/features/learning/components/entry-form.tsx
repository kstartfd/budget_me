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
