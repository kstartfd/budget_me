"use client"

import { useTransition } from "react"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useTheme } from "next-themes"
import { useLocale, useTranslations } from "next-intl"

import { GearIcon } from "@/components/app-icons"
import { useAuth } from "@/features/auth"
import { useMounted } from "@/hooks/use-mounted"
import type { Locale } from "@/i18n/config"
import { setUserLocale } from "@/i18n/locale"
import { cn } from "@/lib/utils"

/**
 * RU/EN toggle, theme toggle and (web accounts only) sign-out. Shared by the
 * mobile header and the desktop top bar so both stay in sync.
 */
export function HeaderControls() {
  const t = useTranslations("app")
  const tNav = useTranslations("nav")
  const locale = useLocale()
  const pathname = usePathname()
  const router = useRouter()
  const [, startTransition] = useTransition()
  const { resolvedTheme, setTheme } = useTheme()
  const mounted = useMounted()
  const { user, logout } = useAuth()
  // Telegram sessions re-auth on every open — sign-out only makes sense for
  // web accounts.
  const webAccount = user != null && user.telegram_id == null

  function switchLocale(next: Locale) {
    if (next === locale) return
    startTransition(async () => {
      await setUserLocale(next)
      router.refresh()
    })
  }

  return (
    <div className="flex items-center gap-2">
      <div className="border-border bg-secondary flex h-[30px] items-center rounded-full border px-1 font-mono">
        {(["ru", "en"] as const).map((l) => (
          <button
            key={l}
            type="button"
            aria-pressed={locale === l}
            onClick={() => switchLocale(l)}
            className={cn(
              "rounded-full px-[9px] py-1 text-[11px] font-semibold uppercase",
              locale === l
                ? "bg-primary text-primary-foreground"
                : "text-faint",
            )}
          >
            {l}
          </button>
        ))}
      </div>
      <Link
        href="/settings"
        aria-label={tNav("settings")}
        aria-current={pathname === "/settings" ? "page" : undefined}
        className={cn(
          "border-border bg-secondary flex size-[30px] items-center justify-center rounded-full border",
          pathname === "/settings" ? "text-primary" : "text-muted-foreground",
        )}
      >
        <GearIcon />
      </Link>
      <button
        type="button"
        aria-label={t("toggleTheme")}
        onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        className="border-border bg-secondary flex size-[30px] items-center justify-center rounded-full border text-[15px]"
      >
        {mounted ? (resolvedTheme === "dark" ? "☾" : "☀") : "☾"}
      </button>
      {webAccount && (
        <button
          type="button"
          aria-label={t("logout")}
          onClick={logout}
          className="border-border bg-secondary text-muted-foreground flex size-[30px] items-center justify-center rounded-full border"
        >
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <path d="M16 17l5-5-5-5M21 12H9" />
          </svg>
        </button>
      )}
    </div>
  )
}
