"use client"

import { useTranslations } from "next-intl"

import { StackIcon } from "@/components/app-icons"
import { Soon } from "@/components/soon"

/** `/lists` — placeholder until the lists feature is built (step 4 of the plan). */
export default function ListsPage() {
  const t = useTranslations("lists")

  return <Soon icon={StackIcon} title={t("title")} description={t("soon")} />
}
