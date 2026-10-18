"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconBrandTelegram, IconCheck, IconMail } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"

import { useAccount } from "../queries"
import { LinkEmailSheet } from "./link-email-sheet"

/**
 * The two login identities. Telegram is the account origin (accounts are created
 * from initData), so it is read-only; the one link action is adding an
 * email+password path so the web works too.
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
        <IdentityRow
          icon={<IconMail size={18} />}
          label={t("emailLabel")}
          value={hasEmail ? (email ?? "") : t("emailNotLinked")}
          linked={hasEmail}
          linkedText={t("linked")}
          cta={
            hasEmail ? null : (
              <Button
                type="button"
                onClick={() => setEmailOpen(true)}
                className="text-primary-foreground mt-3 h-auto w-full rounded-[14px] py-3 text-[13.5px] font-bold shadow-[0_8px_20px_-6px_var(--primary-glow)]"
              >
                <IconMail size={16} />
                {t("addEmail")}
              </Button>
            )
          }
        />

        <IdentityRow
          icon={<IconBrandTelegram size={18} />}
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
        <span className="text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl bg-[color:var(--primary-wash)]">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-[13.5px] font-semibold">{label}</div>
          <div className="text-faint mt-[2px] truncate text-[11.5px]">
            {value}
          </div>
        </div>
        {linked && (
          <span className="flex shrink-0 items-center gap-1 rounded-full bg-[color:var(--success-wash)] px-2.5 py-1 text-[11px] font-semibold text-[color:var(--success)]">
            <IconCheck size={12} />
            {linkedText}
          </span>
        )}
      </div>
      {cta}
    </div>
  )
}
