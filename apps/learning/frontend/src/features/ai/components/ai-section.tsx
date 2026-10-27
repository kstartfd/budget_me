"use client"

import { useState } from "react"
import { useForm, useWatch } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import { IconCheck, IconKey } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

import {
  findModel,
  findProvider,
  providers,
  type ModelSpec,
  type ProviderId,
} from "../catalog"
import { useDeleteAiSettings, useSaveAiSettings } from "../mutations"
import { useAiSettings } from "../queries"
import {
  makeAiFormSchema,
  type AiFormValues,
  type AiSettings,
} from "../schemas"

/**
 * Which agent writes the translation and the explanation of a saved entry:
 * provider, then one of its models, then how hard that model thinks. The last
 * step differs by model ("effort" levels, or DeepSeek's thinking switch), so it
 * is drawn from the catalog, not hard-coded per provider. The key is write-only:
 * once saved the screen only ever sees its last four characters.
 */
export function AiSection() {
  const t = useTranslations("ai")
  const settings = useAiSettings()

  return (
    <section className="flex flex-col gap-[9px]">
      <div className="flex items-center justify-between gap-3 px-0.5">
        <span className="text-faint font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
          {t("kicker")}
        </span>
        {settings.data && (
          <span className="flex shrink-0 items-center gap-1 rounded-full bg-[color:var(--success-wash)] px-2.5 py-1 text-[11px] font-semibold text-[color:var(--success)]">
            <IconCheck size={12} />
            {t("connected")}
          </span>
        )}
      </div>

      {settings.isError ? (
        <p role="alert" className="text-destructive px-0.5 text-xs">
          {t("loadError")}
        </p>
      ) : settings.isPending ? (
        <Skeleton className="h-[330px] rounded-[18px]" />
      ) : (
        <AiForm saved={settings.data} />
      )}

      <p className="text-faint px-0.5 text-[11.5px] leading-[1.4]">
        {t("sectionHint")}
      </p>
    </section>
  )
}

function AiForm({ saved }: { saved: AiSettings | null }) {
  const t = useTranslations("ai")
  const save = useSaveAiSettings()
  const disconnect = useDeleteAiSettings()
  const [replacing, setReplacing] = useState(false)

  const initialProvider = findProvider(saved?.provider ?? "") ?? providers[0]
  const initialModel =
    findModel(initialProvider.id, saved?.model ?? "") ??
    initialProvider.models[0]
  const initialLevel = initialModel.levels.includes(saved?.level ?? "")
    ? (saved?.level ?? initialModel.defaultLevel)
    : initialModel.defaultLevel

  const {
    register,
    handleSubmit,
    control,
    setValue,
    reset,
    formState: { errors, isDirty },
  } = useForm<AiFormValues>({
    resolver: zodResolver(
      makeAiFormSchema({ storedProvider: saved?.provider ?? null, replacing }),
    ),
    defaultValues: {
      provider: initialProvider.id,
      model: initialModel.id,
      level: initialLevel,
      apiKey: "",
    },
  })

  const providerId = useWatch({ control, name: "provider" })
  const modelId = useWatch({ control, name: "model" })
  const level = useWatch({ control, name: "level" })

  const provider = findProvider(providerId) ?? providers[0]
  const model = findModel(provider.id, modelId) ?? provider.models[0]
  const storedKeyHint = saved?.provider === provider.id ? saved.keyHint : null
  const keyRequired = storedKeyHint === null || replacing
  const justSaved = saved !== null && !isDirty && !replacing

  function pickProvider(id: ProviderId) {
    if (id === providerId) return

    const next = findProvider(id) ?? providers[0]
    const first = next.models[0]
    const options = { shouldDirty: true }
    setValue("provider", next.id, options)
    setValue("model", first.id, options)
    setValue("level", first.defaultLevel, options)
    setValue("apiKey", "")
    setReplacing(false)
  }

  function pickModel(next: ModelSpec) {
    const options = { shouldDirty: true }
    setValue("model", next.id, options)

    if (!next.levels.includes(level)) {
      setValue("level", next.defaultLevel, options)
    }
  }

  function onSubmit(values: AiFormValues) {
    save.mutate(
      {
        provider: values.provider,
        model: values.model,
        level: values.level,
        apiKey: values.apiKey || undefined,
      },
      {
        onSuccess: () => {
          setReplacing(false)
          reset({ ...values, apiKey: "" })
        },
      },
    )
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      noValidate
      className="bg-card border-border divide-border divide-y rounded-[18px] border shadow-[var(--shadow-card)]"
    >
      <Block label={t("provider")}>
        <div
          role="radiogroup"
          aria-label={t("provider")}
          className="grid grid-cols-3 gap-2"
        >
          {providers.map((item) => {
            const active = item.id === provider.id

            return (
              <button
                key={item.id}
                type="button"
                role="radio"
                aria-checked={active}
                onClick={() => pickProvider(item.id)}
                className={cn(
                  "focus-visible:ring-primary flex min-w-0 flex-col items-start gap-2 rounded-[14px] border p-3 text-left transition-colors outline-none focus-visible:ring-2",
                  active
                    ? "border-primary bg-[color:var(--primary-wash)]"
                    : "border-border hover:bg-secondary/60",
                )}
              >
                <span
                  className={cn(
                    "flex size-[34px] items-center justify-center rounded-full text-[14px] font-bold",
                    active
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-muted-foreground",
                  )}
                >
                  {item.badge}
                </span>
                <span className="min-w-0">
                  <span className="block truncate text-[13.5px] font-semibold">
                    {item.label}
                  </span>
                  <span className="text-faint mt-px block truncate text-[11px]">
                    {t(`api.${item.id}`)}
                  </span>
                </span>
              </button>
            )
          })}
        </div>
      </Block>

      <Block label={t("agent")}>
        <div
          role="group"
          aria-label={t("agent")}
          className="flex flex-wrap gap-2"
        >
          {provider.models.map((item) => (
            <Pill
              key={item.id}
              active={item.id === model.id}
              onClick={() => pickModel(item)}
            >
              {item.label}
            </Pill>
          ))}
        </div>
        <code className="text-faint mt-2.5 block font-mono text-[11px]">
          {model.id}
        </code>
      </Block>

      <Block label={t(model.kind)}>
        <div
          role="group"
          aria-label={t(model.kind)}
          className="flex flex-wrap gap-2"
        >
          {model.levels.map((item) => (
            <Pill
              key={item}
              active={item === level}
              onClick={() => setValue("level", item, { shouldDirty: true })}
            >
              {t(`levels.${item}`)}
            </Pill>
          ))}
        </div>
        <p className="text-muted-foreground mt-2.5 text-[11.5px] leading-[1.4]">
          {t(`${model.kind}Hint`)}
        </p>
        {errors.level && (
          <p role="alert" className="text-destructive mt-2 text-xs">
            {t(`errors.${errors.level.message}`)}
          </p>
        )}
      </Block>

      <Block label={t("apiKey")}>
        {keyRequired ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ai-key" className="sr-only">
              {t("apiKey")}
            </Label>
            <Input
              id="ai-key"
              type="password"
              autoComplete="off"
              spellCheck={false}
              placeholder={provider.keyPlaceholder}
              aria-invalid={errors.apiKey ? true : undefined}
              className="h-11 rounded-xl px-3.5 font-mono text-[13px]"
              {...register("apiKey")}
            />
            {errors.apiKey && (
              <p role="alert" className="text-destructive text-xs">
                {t(`errors.${errors.apiKey.message}`)}
              </p>
            )}
            {replacing && (
              <button
                type="button"
                onClick={() => {
                  setReplacing(false)
                  setValue("apiKey", "")
                }}
                className="text-muted-foreground self-start text-xs font-semibold"
              >
                {t("keepKey")}
              </button>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <span className="text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl bg-[color:var(--primary-wash)]">
              <IconKey size={18} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-[13.5px] font-semibold">{t("keySaved")}</div>
              <div className="text-faint mt-[2px] truncate font-mono text-[11.5px]">
                ••••{storedKeyHint}
              </div>
            </div>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setReplacing(true)}
              className="h-auto shrink-0 rounded-xl px-3.5 py-2 text-[12.5px] font-semibold"
            >
              {t("replaceKey")}
            </Button>
          </div>
        )}
      </Block>

      <div className="p-[14px]">
        <Button
          type="submit"
          disabled={save.isPending || justSaved}
          className="text-primary-foreground h-auto w-full rounded-[14px] py-3 text-[13.5px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
        >
          {save.isPending ? (
            t("saving")
          ) : justSaved ? (
            <>
              <IconCheck size={16} />
              {t("saved")}
            </>
          ) : (
            t("save")
          )}
        </Button>
        {(save.isError || disconnect.isError) && (
          <p role="alert" className="text-destructive mt-3 text-xs">
            {(save.isError && save.error.message) || t("saveFailed")}
          </p>
        )}
        {saved && (
          <Button
            type="button"
            variant="ghost"
            disabled={disconnect.isPending}
            onClick={() =>
              disconnect.mutate(undefined, {
                onSuccess: () => setReplacing(false),
              })
            }
            className="text-muted-foreground mt-1.5 h-auto w-full rounded-[14px] py-2.5 text-[12.5px] font-semibold"
          >
            {disconnect.isPending ? t("disconnecting") : t("disconnect")}
          </Button>
        )}
      </div>
    </form>
  )
}

function Block({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="p-[14px]">
      <div className="text-muted-foreground mb-2.5 text-[11.5px] font-semibold">
        {label}
      </div>
      {children}
    </div>
  )
}

function Pill({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "h-auto rounded-full px-4 py-2 text-[13px] font-medium",
        active ? "bg-primary/15 text-primary" : "text-foreground/75",
      )}
    >
      {children}
    </Button>
  )
}
