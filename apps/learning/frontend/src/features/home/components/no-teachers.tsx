"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

import { setupSteps } from "../profile"
import { Caption, card } from "./parts"

/** The first thing someone with no language sees: what a language is, and how to begin. */
export function NoTeachers() {
  const t = useTranslations("home")

  return (
    <section className={cn(card, "flex max-w-[880px] flex-col gap-4 p-6 lg:p-8")}>
      <div>
        <h2 className="mb-1.5 text-xl font-semibold">{t("firstTitle")}</h2>
        <p className="text-muted-foreground max-w-[560px] text-sm">{t("firstText")}</p>
      </div>
      <div>
        <Button asChild size="lg" className="h-11 px-5 text-sm">
          <Link href="/learning/start">{t("firstCta")}</Link>
        </Button>
      </div>
      <div className="border-border border-t pt-4">
        <Caption className="mb-2 block">{t("howTitle")}</Caption>
        {/* Clip the divider before the first step of each grid row. */}
        <ol className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-3 overflow-hidden">
          {setupSteps.map((id, index) => (
            <li
              key={id}
              className="relative text-[13px] before:absolute before:inset-y-0 before:-start-1.5 before:border-s before:border-border"
            >
              <p className="font-medium">
                {index + 1}. {t(`step.${id}`)}
              </p>
              <p className="text-muted-foreground">{t(`stepHint.${id}`)}</p>
            </li>
          ))}
        </ol>
        <p className="text-muted-foreground mt-6 text-xs">{t("howNote")}</p>
      </div>
    </section>
  )
}
