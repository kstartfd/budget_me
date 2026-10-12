"use client"

import { useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import { useForm } from "react-hook-form"

import { WalletLogoIcon } from "@/components/app-icons"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { ApiError } from "../api"
import { useAuth } from "../auth-context"
import { loginFormSchema, type LoginFormValues } from "../schemas"

/**
 * The gate for visitors outside Telegram (and the fallback when Telegram
 * auto-login fails): email/password sign-in only. New accounts are created
 * exclusively inside the Telegram Mini App (identity comes from initData), so
 * there is no web sign-up — newcomers are pointed at the bot. The app itself
 * stays hidden until a session exists.
 */
export function AuthScreen() {
  const t = useTranslations("auth")
  const { bootError, retryTelegram } = useAuth()

  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-6 px-[18px] py-10">
      <div className="flex flex-col items-center gap-3 text-center">
        <div className="bg-primary text-primary-foreground flex size-12 items-center justify-center rounded-2xl">
          <WalletLogoIcon size={24} />
        </div>
        <div>
          <h1 className="text-[22px] font-bold tracking-[-0.02em]">
            {t("appTitle")}
          </h1>
          <p className="text-muted-foreground mt-1 text-[13.5px]">
            {t("tagline")}
          </p>
        </div>
      </div>

      {bootError && (
        <div className="border-border bg-card rounded-2xl border p-3 text-center">
          <p className="text-destructive text-[12.5px] font-medium">
            {t("telegramFailed")}
          </p>
          <p className="text-faint mt-0.5 font-mono text-[11.5px]">{bootError}</p>
          <button
            type="button"
            onClick={retryTelegram}
            className="text-primary mt-2 text-[12.5px] font-semibold"
          >
            {t("retry")}
          </button>
        </div>
      )}

      <div className="bg-card border-border rounded-[22px] border p-5 shadow-[var(--shadow-card)]">
        <LoginForm />
      </div>

      <p className="text-faint text-center text-[12.5px]">
        {t("orTelegram")}{" "}
        <a
          href="https://t.me/money_carrot_bot"
          className="text-primary font-semibold"
        >
          @money_carrot_bot
        </a>
      </p>
    </main>
  )
}

function fieldError(
  t: ReturnType<typeof useTranslations<"auth">>,
  key: string | undefined,
) {
  if (!key) return null
  return (
    <p role="alert" className="text-destructive text-xs">
      {t(`errors.${key}` as Parameters<typeof t>[0])}
    </p>
  )
}

function LoginForm() {
  const t = useTranslations("auth")
  const { login } = useAuth()
  const [serverError, setServerError] = useState<string | null>(null)

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginFormSchema),
    defaultValues: { email: "", password: "" },
  })

  async function onSubmit(values: LoginFormValues) {
    setServerError(null)
    try {
      await login(values.email, values.password)
    } catch (err) {
      setServerError(err instanceof ApiError ? err.message : t("networkError"))
    }
  }

  return (
    <form
      onSubmit={form.handleSubmit(onSubmit)}
      className="flex flex-col gap-3.5"
      noValidate
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-email">{t("email")}</Label>
        <Input
          id="login-email"
          type="email"
          autoComplete="email"
          inputMode="email"
          {...form.register("email")}
        />
        {fieldError(t, form.formState.errors.email?.message)}
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-password">{t("password")}</Label>
        <Input
          id="login-password"
          type="password"
          autoComplete="current-password"
          {...form.register("password")}
        />
        {fieldError(t, form.formState.errors.password?.message)}
      </div>
      {serverError && (
        <p role="alert" className="text-destructive text-[12.5px] font-medium">
          {serverError}
        </p>
      )}
      <button
        type="submit"
        disabled={form.formState.isSubmitting}
        className="bg-primary text-primary-foreground mt-1 w-full rounded-2xl p-3.5 text-[14.5px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] disabled:opacity-60"
      >
        {form.formState.isSubmitting ? t("signingIn") : t("signIn")}
      </button>
    </form>
  )
}
