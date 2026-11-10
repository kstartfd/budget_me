"use client"

import { useTransition } from "react"
import { useRouter } from "next/navigation"
import { useTheme } from "next-themes"
import { useLocale, useTranslations } from "next-intl"
import { IconLogout, IconMoon, IconSun } from "@tabler/icons-react"

import { useAuth } from "@/features/auth"
import { useMounted } from "@/hooks/use-mounted"
import type { Locale } from "@/i18n/config"
import { setUserLocale } from "@/i18n/locale"
import { cn } from "@/lib/utils"

const iconButton =
  "border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground flex size-9 shrink-0 items-center justify-center rounded-md border transition-colors"

/** RU/EN segmented control: a muted track with the active language raised. */
export function LocaleSwitch({ className }: { className?: string }) {
  const locale = useLocale()
  const router = useRouter()
  const [, startTransition] = useTransition()

  function switchLocale(next: Locale) {
    if (next === locale) return
    startTransition(async () => {
      await setUserLocale(next)
      router.refresh()
    })
  }

  return (
    <div
      className={cn(
        "bg-muted flex h-9 items-center gap-0.5 rounded-md p-1",
        className,
      )}
    >
      {(["ru", "en"] as const).map((l) => (
        <button
          key={l}
          type="button"
          aria-pressed={locale === l}
          onClick={() => switchLocale(l)}
          className={cn(
            "h-full flex-1 rounded-sm px-3 text-[13px] font-medium uppercase transition-colors",
            locale === l
              ? "bg-background text-foreground"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {l}
        </button>
      ))}
    </div>
  )
}

export function ThemeButton({ className }: { className?: string }) {
  const t = useTranslations("app")
  const { resolvedTheme, setTheme } = useTheme()
  const mounted = useMounted()
  const dark = mounted && resolvedTheme === "dark"

  return (
    <button
      type="button"
      aria-label={t("toggleTheme")}
      onClick={() => setTheme(dark ? "light" : "dark")}
      className={cn(iconButton, className)}
    >
      {dark ? <IconMoon size={16} /> : <IconSun size={16} />}
    </button>
  )
}

/** Sign-out for web accounts; Telegram sessions re-auth on every open. */
export function LogoutButton({ className }: { className?: string }) {
  const t = useTranslations("app")
  const { user, logout } = useAuth()

  if (user == null || user.telegram_id != null) return null

  return (
    <button
      type="button"
      aria-label={t("logout")}
      onClick={logout}
      className={cn(iconButton, className)}
    >
      <IconLogout size={16} />
    </button>
  )
}

/** Mobile header controls; the desktop sidebar lays the same pieces out itself. */
export function HeaderControls() {
  return (
    <div className="flex items-center gap-2">
      <LocaleSwitch />
      <ThemeButton />
      <LogoutButton />
    </div>
  )
}
