"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useTranslations } from "next-intl"
import { useForm } from "react-hook-form"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import { linkEmailFormSchema, type LinkEmailFormValues } from "../schemas"

export function LinkEmailForm({
  submitting,
  error,
  onSubmit,
  onCancel,
}: {
  submitting: boolean
  error: string | null
  onSubmit: (values: LinkEmailFormValues) => void
  onCancel: () => void
}) {
  const t = useTranslations("account")
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LinkEmailFormValues>({
    resolver: zodResolver(linkEmailFormSchema),
    defaultValues: { email: "", password: "" },
  })

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col gap-3.5"
      noValidate
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="link-email">{t("email")}</Label>
        <Input
          id="link-email"
          type="email"
          autoComplete="email"
          inputMode="email"
          autoFocus
          aria-invalid={errors.email ? true : undefined}
          {...register("email")}
        />
        {fieldError(t, errors.email?.message)}
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="link-password">{t("password")}</Label>
        <Input
          id="link-password"
          type="password"
          autoComplete="new-password"
          aria-invalid={errors.password ? true : undefined}
          {...register("password")}
        />
        {fieldError(t, errors.password?.message)}
      </div>

      {error && (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      )}

      <div className="mt-1 flex gap-2.5">
        <Button
          type="submit"
          disabled={submitting}
          className="text-primary-foreground h-auto flex-1 rounded-[18px] p-[15px] text-[15px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
        >
          {submitting ? t("saving") : t("save")}
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={onCancel}
          disabled={submitting}
          className="text-muted-foreground h-auto rounded-[18px] px-5 py-[15px] text-sm font-semibold"
        >
          {t("cancel")}
        </Button>
      </div>
    </form>
  )
}

function fieldError(
  t: ReturnType<typeof useTranslations<"account">>,
  key: string | undefined,
) {
  if (!key) return null

  return (
    <p role="alert" className="text-destructive text-xs">
      {t(`errors.${key}` as Parameters<typeof t>[0])}
    </p>
  )
}
