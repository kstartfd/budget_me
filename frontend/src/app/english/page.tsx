"use client"

import { useTranslations } from "next-intl"

import { LanguageIcon } from "@/components/app-icons"
import { Soon } from "@/components/soon"

/** `/english` — placeholder until the phrases feature is built (step 3 of the plan). */
export default function EnglishPage() {
  const t = useTranslations("english")

  return <Soon icon={LanguageIcon} title={t("title")} description={t("soon")} />
}
