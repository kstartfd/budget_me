"use client"

import { useState } from "react"
import Link from "next/link"
import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import {
  IconArrowLeft,
  IconCheck,
  IconCopy,
  IconKey,
  IconPlus,
} from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { copyToClipboard } from "@/lib/clipboard"
import { useLearning, useLearningTokens } from "../hooks"
import {
  tokenInputSchema,
  type LearningToken,
  type TokenInput,
} from "../schemas"
import { LearningModal } from "./learning-modal"
import { SecretField } from "./secret-field"

/** `bml_` + 43 base64url characters, as issued by the backend. */
const TOKEN_LENGTH = 47

function maskedToken(prefix: string, length = TOKEN_LENGTH) {
  return prefix + "•".repeat(Math.max(length - prefix.length, 0))
}

/** `/api/v1/learning/captures` on this origin — the one endpoint every
 * extension token posts to; the token itself picks the topic server-side. */
function captureEndpoint() {
  if (typeof window === "undefined") return "/api/v1/learning/captures"
  return `${window.location.origin}/api/v1/learning/captures`
}

export function TokenScreen() {
  const t = useTranslations("learning")
  const { topics } = useLearning()
  const { tokens, create, revoke, reveal } = useLearningTokens()
  const [creating, setCreating] = useState(false)
  const [secret, setSecret] = useState<string | null>(null)
  const [endpointCopied, setEndpointCopied] = useState(false)
  const [copyError, setCopyError] = useState(false)
  const [revoking, setRevoking] = useState<LearningToken | null>(null)
  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<TokenInput>({
    resolver: zodResolver(tokenInputSchema),
    defaultValues: { name: "", scope: "capture", topicId: "" },
  })

  function topicName(topicId: string | null) {
    if (!topicId) return t("tokenAllTopics")
    return topics.data?.find((topic) => topic.id === topicId)?.name ?? topicId
  }

  async function copyEndpoint() {
    const copied = await copyToClipboard(captureEndpoint())
    setEndpointCopied(copied)
    setCopyError(!copied)
  }
  function hideSecret() {
    setSecret(null)
    setEndpointCopied(false)
    setCopyError(false)
    create.reset()
  }
  return (
    <div className="anim-fade max-w-2xl space-y-[18px]">
      <header>
        <Link
          href="/learning"
          className="text-muted-foreground mb-2 inline-flex items-center gap-1 text-xs"
        >
          <IconArrowLeft size={14} />
          {t("title")}
        </Link>
        <h1 className="text-[22px] font-bold tracking-tight">{t("tokens")}</h1>
        <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
          {t("tokensHint")}
        </p>
      </header>

      <div className="bg-card border-border rounded-[22px] border p-4 shadow-[var(--shadow-card)]">
        <p className="text-faint mb-1.5 text-[11px] font-semibold tracking-[0.06em] uppercase">
          {t("tokenEndpoint")}
        </p>
        <div className="flex items-center gap-2">
          <code className="bg-secondary text-foreground min-w-0 flex-1 truncate rounded-xl px-3.5 py-3 font-mono text-xs">
            {captureEndpoint()}
          </code>
          <Button
            type="button"
            variant="secondary"
            size="icon"
            aria-label={t("copyEndpoint")}
            className="h-11 w-11 shrink-0 rounded-xl"
            onClick={() => void copyEndpoint()}
          >
            {endpointCopied ? <IconCheck size={17} /> : <IconCopy size={17} />}
          </Button>
        </div>
        <p className="text-muted-foreground mt-2 text-xs leading-relaxed">
          {t("tokenEndpointHint")}
        </p>
      </div>

      <Button
        className="text-primary-foreground h-auto rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
        disabled={!topics.data?.length}
        onClick={() => {
          reset({
            name: "",
            scope: "capture",
            topicId: topics.data?.[0]?.id ?? "",
          })
          create.reset()
          setCreating(true)
        }}
      >
        <IconPlus size={17} />
        {t("createToken")}
      </Button>
      {!topics.isPending && topics.data?.length === 0 && (
        <p className="text-muted-foreground text-xs leading-relaxed">
          {t("tokenNoTopics")}
        </p>
      )}
      {tokens.isPending ? (
        <Skeleton className="h-24 w-full rounded-[22px]" />
      ) : tokens.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("loadError")}
        </p>
      ) : (
        <div className="space-y-3">
          {tokens.data?.map((token) => (
            <div
              key={token.id}
              className="bg-card border-border space-y-3 rounded-[22px] border p-4 shadow-[var(--shadow-card)]"
            >
              <div className="flex items-center gap-3">
                <span className="bg-secondary text-primary flex size-10 shrink-0 items-center justify-center rounded-xl">
                  <IconKey size={20} />
                </span>
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-sm font-semibold">
                    {token.name}
                  </h2>
                  <p className="text-muted-foreground mt-1 truncate text-xs">
                    {t(token.scope)} · {topicName(token.topicId)}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  className="text-destructive shrink-0 text-xs"
                  onClick={() => {
                    revoke.reset()
                    setRevoking(token)
                  }}
                >
                  {t("revoke")}
                </Button>
              </div>
              {token.revealable ? (
                <SecretField
                  compact
                  mask={maskedToken(token.prefix)}
                  load={() => reveal(token.id)}
                />
              ) : (
                <div>
                  <p className="text-faint font-mono text-xs">
                    {token.prefix}…
                  </p>
                  <p className="text-muted-foreground mt-1 text-xs leading-relaxed">
                    {t("tokenLegacy")}
                  </p>
                </div>
              )}
            </div>
          ))}
          {tokens.data?.length === 0 && (
            <div className="bg-card border-border rounded-[22px] border p-5">
              <p className="text-sm font-semibold">{t("tokensEmpty")}</p>
              <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
                {t("tokensEmptyHint")}
              </p>
            </div>
          )}
        </div>
      )}
      <p className="text-faint text-xs leading-relaxed">
        {t("tokensRestriction")}
      </p>

      {creating && (
        <LearningModal
          title={t("createToken")}
          onClose={() => {
            if (!create.isPending) setCreating(false)
          }}
        >
          <form
            className="space-y-4"
            onSubmit={handleSubmit((values) =>
              create.mutate(values, {
                onSuccess: (data) => {
                  setSecret(data.token)
                  setCreating(false)
                  create.reset()
                },
              }),
            )}
          >
            <label className="block space-y-2 text-sm font-medium">
              <span>{t("tokenName")}</span>
              <Input
                autoFocus
                maxLength={80}
                className="bg-secondary border-border focus-visible:border-primary h-auto rounded-2xl px-4 py-3.5 text-[15px] focus-visible:ring-0"
                placeholder={t("tokenPlaceholder")}
                aria-invalid={!!errors.name}
                {...register("name")}
              />
            </label>
            <div className="space-y-2 text-sm font-medium">
              <span>{t("tokenTopic")}</span>
              <Controller
                control={control}
                name="topicId"
                render={({ field }) => (
                  <div
                    role="radiogroup"
                    aria-label={t("tokenTopic")}
                    className="flex flex-wrap gap-2"
                  >
                    {topics.data?.map((topic) => (
                      <button
                        key={topic.id}
                        type="button"
                        role="radio"
                        aria-checked={field.value === topic.id}
                        onClick={() => field.onChange(topic.id)}
                        className={`rounded-full border px-3.5 py-2 text-xs font-semibold transition-colors ${
                          field.value === topic.id
                            ? "border-primary/40 bg-primary/10 text-primary"
                            : "border-border bg-secondary text-muted-foreground"
                        }`}
                      >
                        {topic.name}
                      </button>
                    ))}
                  </div>
                )}
              />
              <p className="text-muted-foreground text-xs leading-relaxed">
                {t("tokenTopicHint")}
              </p>
            </div>
            <Controller
              control={control}
              name="scope"
              render={({ field }) => (
                <div
                  role="group"
                  aria-label={t("tokenScope")}
                  className="space-y-2"
                >
                  {(["capture", "read_write"] as const).map((scope) => (
                    <Button
                      key={scope}
                      type="button"
                      variant="outline"
                      aria-pressed={field.value === scope}
                      onClick={() => field.onChange(scope)}
                      className={`border-border h-auto w-full justify-start rounded-2xl p-4 text-left whitespace-normal ${field.value === scope ? "border-primary/40 bg-primary/10" : ""}`}
                    >
                      <span>
                        <span className="block text-sm font-semibold">
                          {t(scope)}
                        </span>
                        <span className="text-muted-foreground mt-1 block text-xs leading-relaxed">
                          {t(`${scope}Hint`)}
                        </span>
                      </span>
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
            {create.isError && (
              <p role="alert" className="text-destructive text-sm">
                {t("saveError")}
              </p>
            )}
            <Button
              type="submit"
              className="text-primary-foreground h-auto w-full rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
              disabled={create.isPending}
            >
              {t(create.isPending ? "saving" : "createToken")}
            </Button>
          </form>
        </LearningModal>
      )}
      {secret && (
        <LearningModal
          title={t("tokenReady")}
          description={t("tokenOnce")}
          onClose={hideSecret}
        >
          <div className="space-y-4">
            <div className="space-y-2">
              <span className="text-sm font-medium">{t("tokenValue")}</span>
              <SecretField
                mask={maskedToken(secret.slice(0, 12), secret.length)}
                load={() => Promise.resolve(secret)}
              />
            </div>
            <div className="space-y-2">
              <span className="text-sm font-medium">{t("tokenEndpoint")}</span>
              <div className="flex items-center gap-2">
                <code className="bg-secondary text-foreground min-w-0 flex-1 truncate rounded-2xl px-4 py-3.5 font-mono text-xs">
                  {captureEndpoint()}
                </code>
                <Button
                  type="button"
                  variant="secondary"
                  size="icon"
                  aria-label={t("copyEndpoint")}
                  className="h-[52px] w-[52px] shrink-0 rounded-2xl"
                  onClick={() => void copyEndpoint()}
                >
                  {endpointCopied ? (
                    <IconCheck size={18} />
                  ) : (
                    <IconCopy size={18} />
                  )}
                </Button>
              </div>
            </div>
            {copyError && (
              <p role="alert" className="text-destructive text-sm">
                {t("copyError")}
              </p>
            )}
            <Button
              variant="ghost"
              className="w-full text-sm font-semibold"
              onClick={hideSecret}
            >
              {t("done")}
            </Button>
          </div>
        </LearningModal>
      )}
      {revoking && (
        <LearningModal
          title={t("revoke")}
          description={t("revokeWarning", { name: revoking.name })}
          onClose={() => {
            if (!revoke.isPending) setRevoking(null)
          }}
        >
          <div className="flex gap-3">
            <Button
              variant="secondary"
              className="flex-1"
              disabled={revoke.isPending}
              onClick={() => setRevoking(null)}
            >
              {t("cancel")}
            </Button>
            <Button
              variant="destructive"
              className="flex-1"
              disabled={revoke.isPending}
              onClick={() =>
                revoke.mutate(revoking.id, {
                  onSuccess: () => setRevoking(null),
                })
              }
            >
              {t("revoke")}
            </Button>
          </div>
          {revoke.isError && (
            <p role="alert" className="text-destructive text-sm">
              {t("deleteError")}
            </p>
          )}
        </LearningModal>
      )}
    </div>
  )
}
