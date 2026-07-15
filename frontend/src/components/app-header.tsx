"use client"

import { useLocale, useTranslations } from "next-intl"

import { WalletLogoIcon } from "@/components/app-icons"
import { HeaderControls } from "@/components/header-controls"
import { useToday } from "@/features/settings"
import { currentMonthKey, monthTitle } from "@/lib/month"

/**
 * Mobile header (phone / Telegram): accent logo chip, app name over the
 * current month, plus the shared locale/theme/sign-out controls. Hidden on
 * desktop, where the left sidebar (DesktopNav) takes over.
 */
export function AppHeader() {
  const t = useTranslations("app")
  const locale = useLocale()
  const today = useToday()

  return (
    <header className="bg-background border-border sticky top-0 z-20 border-b lg:hidden">
      <div className="mx-auto flex w-full max-w-md items-center gap-3 px-5 pt-4 pb-2.5">
        <div className="bg-primary text-primary-foreground flex size-[34px] shrink-0 items-center justify-center rounded-[11px]">
          <WalletLogoIcon />
        </div>
        <div className="flex-1">
          <div className="text-[15px] font-semibold tracking-[-0.01em]">
            {t("name")}
          </div>
          <div className="text-faint font-mono text-[11.5px] tracking-[0.02em]">
            {monthTitle(currentMonthKey(today), locale)}
          </div>
        </div>
        <HeaderControls />
      </div>
    </header>
  )
}
