"use client"

import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useLearning } from "../hooks"
import { topicInputSchema, type Topic, type TopicInput } from "../schemas"
import { LearningModal } from "./learning-modal"

export function TopicForm({
  topic,
  onClose,
}: {
  topic?: Topic
  onClose: () => void
}) {
  const t = useTranslations("learning")
  const { saveTopic } = useLearning()
  const locale = useLocale()
  const languages = [
    ...new Set([
      "en",
      "en-US",
      "es",
      "fr",
      "de",
      "it",
      "pt",
      "pl",
      "ru",
      "be",
      "uk",
      "zh",
      "ja",
      "ko",
      "tr",
      "ar",
      "hi",
      "nl",
      "sv",
      "no",
      "da",
      "fi",
      "cs",
      "el",
      "he",
      ...(topic ? [topic.language, topic.translationLanguage] : []),
    ]),
  ]
  const languageNames = new Intl.DisplayNames([locale], { type: "language" })
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<TopicInput>({
    resolver: zodResolver(topicInputSchema),
    defaultValues: topic ?? {
      name: "",
      language: "en",
      translationLanguage: "ru",
    },
  })
  return (
    <LearningModal
      title={t(topic ? "editTopic" : "newTopic")}
      description={t("topicHint")}
      onClose={onClose}
    >
      <form
        className="space-y-4"
        onSubmit={handleSubmit((values) =>
          saveTopic.mutate(
            { ...values, id: topic?.id },
            { onSuccess: onClose },
          ),
        )}
      >
        <label className="block space-y-2 text-sm font-medium">
          <span>{t("topicName")}</span>
          <Input
            autoFocus
            className="bg-secondary h-12 rounded-2xl"
            placeholder={t("topicPlaceholder")}
            maxLength={80}
            aria-invalid={!!errors.name}
            {...register("name")}
          />
        </label>
        <div className="grid gap-3 sm:grid-cols-2">
          {(["language", "translationLanguage"] as const).map((name) => (
            <div key={name} className="space-y-2 text-sm font-medium">
              <label htmlFor={`topic-${name}`}>{t(name)}</label>
              <Controller
                control={control}
                name={name}
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger
                      id={`topic-${name}`}
                      className="bg-secondary h-12 w-full rounded-2xl"
                      aria-invalid={!!errors[name]}
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {languages.map((code) => (
                        <SelectItem key={code} value={code}>
                          {languageNames.of(code) ?? code}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
            </div>
          ))}
        </div>
        {Object.keys(errors).length > 0 && (
          <p role="alert" className="text-destructive text-sm">
            {t("checkFields")}
          </p>
        )}
        {saveTopic.isError && (
          <p role="alert" className="text-destructive text-sm">
            {t("saveError")}
          </p>
        )}
        <Button
          type="submit"
          disabled={saveTopic.isPending}
          className="h-12 w-full rounded-2xl text-sm font-bold"
        >
          {t(saveTopic.isPending ? "saving" : "save")}
        </Button>
      </form>
    </LearningModal>
  )
}
