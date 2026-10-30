"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"
import { IconArrowLeft } from "@tabler/icons-react"

import { AiSection } from "@/features/ai"

import { AccountSection } from "./account-section"
import { DeleteAccountSection } from "./delete-account-section"
import { ExtensionSection } from "./extension-section"

export function SettingsScreen() {
  const t = useTranslations("learning")

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
        <h1 className="text-[22px] font-bold tracking-tight">{t("settings")}</h1>
      </header>

      <AccountSection />
      <AiSection />
      <ExtensionSection />
      <DeleteAccountSection />
    </div>
  )
}
