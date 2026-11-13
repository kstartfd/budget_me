"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { IconBrandTelegram, IconMail } from "@tabler/icons-react"

import {
  SectionKicker,
  sectionCard,
  sectionHint,
  StatusPill,
} from "@/components/section"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

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
    <section className="flex flex-col gap-3">
      <SectionKicker>{t("kicker")}</SectionKicker>

      <div className={cn(sectionCard, "divide-border divide-y")}>
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
                className="mt-3 h-11 w-full rounded-[10px] text-sm font-semibold"
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

      <p className={sectionHint}>{t("sectionHint")}</p>

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
        <span className="bg-muted text-foreground/75 flex size-[38px] shrink-0 items-center justify-center rounded-[9px]">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold">{label}</div>
          <div className="text-muted-foreground mt-1 truncate text-xs">
            {value}
          </div>
        </div>
        {linked && <StatusPill>{linkedText}</StatusPill>}
      </div>
      {cta}
    </div>
  )
}
