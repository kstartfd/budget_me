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
import { useLearningTokens } from "../hooks"
import {
  tokenInputSchema,
  type LearningToken,
  type TokenInput,
} from "../schemas"
import { LearningModal } from "./learning-modal"

export function TokenScreen() {
  const t = useTranslations("learning")
  const { tokens, create, revoke } = useLearningTokens()
  const [creating, setCreating] = useState(false)
  const [secret, setSecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
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
    defaultValues: { name: "", scope: "capture" },
  })

  async function copy() {
    try {
      if (!secret) return
      await navigator.clipboard.writeText(secret)
      setCopied(true)
      setCopyError(false)
    } catch {
      setCopyError(true)
    }
  }
  function hideSecret() {
    setSecret(null)
    setCopied(false)
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
      <Button
        className="h-11 rounded-2xl"
        onClick={() => {
          reset({ name: "", scope: "capture" })
          create.reset()
          setCreating(true)
        }}
      >
        <IconPlus size={17} />
        {t("createToken")}
      </Button>
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
              className="bg-card border-border flex items-center gap-3 rounded-[22px] border p-4 shadow-[var(--shadow-card)]"
            >
              <span className="bg-secondary text-primary flex size-10 shrink-0 items-center justify-center rounded-xl">
                <IconKey size={20} />
              </span>
              <div className="min-w-0 flex-1">
                <h2 className="truncate text-sm font-semibold">{token.name}</h2>
                <p className="text-faint mt-1 font-mono text-xs">
                  {token.prefix}…
                </p>
                <p className="text-muted-foreground mt-1 text-xs">
                  {t(token.scope)}
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
                className="bg-secondary h-12 rounded-2xl"
                placeholder={t("tokenPlaceholder")}
                aria-invalid={!!errors.name}
                {...register("name")}
              />
            </label>
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
                      className={`h-auto w-full justify-start rounded-2xl p-4 text-left whitespace-normal ${field.value === scope ? "border-primary/40 bg-primary/10" : ""}`}
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
            {errors.name && (
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
              className="h-12 w-full rounded-2xl"
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
            <Input
              aria-label={t("tokenValue")}
              readOnly
              value={secret}
              onFocus={(e) => e.target.select()}
              className="bg-secondary h-12 rounded-2xl font-mono text-xs"
            />
            <Button
              className="h-12 w-full rounded-2xl"
              onClick={() => void copy()}
            >
              {copied ? <IconCheck size={17} /> : <IconCopy size={17} />}
              {t(copied ? "copied" : "copyToken")}
            </Button>
            {copyError && (
              <p role="alert" className="text-destructive text-sm">
                {t("copyError")}
              </p>
            )}
            <Button variant="ghost" className="w-full" onClick={hideSecret}>
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
