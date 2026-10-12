"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"

import { CheckIcon, MailIcon, TelegramIcon } from "@/components/app-icons"

import { useAccount } from "../queries"
import { LinkEmailSheet } from "./link-email-sheet"

/**
 * «Аккаунт» settings section: the two login identities. Telegram is the account
 * origin (every account is created from initData), so it is always present and
 * shown read-only. The only link action is adding an email+password path — a
 * verified form from the authenticated session — so you can also sign in on the
 * web. At least one identity always remains; no unlink action is offered.
 */
export function AccountSection() {
  const t = useTranslations("account")
  const { email, username, telegramId, hasEmail, hasTelegram } = useAccount()

  const [emailOpen, setEmailOpen] = useState(false)

  const telegramValue = hasTelegram
    ? username
      ? `@${username}`
      : `ID ${telegramId}`
    : t("telegramNotLinked")

  return (
    <section className="flex flex-col gap-[9px]">
      <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
        {t("kicker")}
      </span>

      <div className="bg-card border-border divide-border divide-y rounded-[18px] border shadow-[var(--shadow-card)]">
        {/* Email / password identity — the one linkable path */}
        <IdentityRow
          icon={<MailIcon />}
          label={t("emailLabel")}
          value={hasEmail ? (email ?? "") : t("emailNotLinked")}
          linked={hasEmail}
          linkedText={t("linked")}
          cta={
            hasEmail ? null : (
              <CtaButton
                icon={<MailIcon size={16} />}
                label={t("addEmail")}
                onClick={() => setEmailOpen(true)}
              />
            )
          }
        />

        {/* Telegram identity — the account origin, always present, read-only */}
        <IdentityRow
          icon={<TelegramIcon />}
          label={t("telegramLabel")}
          value={telegramValue}
          linked={hasTelegram}
          linkedText={t("linked")}
          cta={null}
        />
      </div>

      <p className="text-faint px-0.5 text-[11.5px] leading-[1.4]">
        {t("sectionHint")}
      </p>

      {emailOpen && <LinkEmailSheet onClose={() => setEmailOpen(false)} />}
    </section>
  )
}

function IdentityRow({
  icon,
  label,
  value,
  linked,
  linkedText,
  cta,
}: {
  icon: React.ReactNode
  label: string
  value: string
  linked: boolean
  linkedText: string
  cta: React.ReactNode
}) {
  return (
    <div className="p-[14px]">
      <div className="flex items-center gap-3">
        <span className="bg-[color:var(--primary-wash)] text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-[13.5px] font-semibold">{label}</div>
          <div className="text-faint mt-[2px] truncate text-[11.5px]">
            {value}
          </div>
        </div>
        {linked && (
          <span className="bg-[color:var(--success-wash)] text-[color:var(--success)] flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-semibold">
            <CheckIcon size={12} />
            {linkedText}
          </span>
        )}
      </div>
      {cta}
    </div>
  )
}

function CtaButton({
  icon,
  label,
  onClick,
  disabled,
}: {
  icon: React.ReactNode
  label: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="bg-primary text-primary-foreground mt-3 flex w-full items-center justify-center gap-2 rounded-[14px] py-3 text-[13.5px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)] transition-opacity disabled:opacity-60"
    >
      {icon}
      {label}
    </button>
  )
}
