"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"
import {
  IconChalkboardTeacher,
  IconMessageDots,
  IconSchool,
  IconSettings,
  IconStack2,
} from "@tabler/icons-react"
import { FullScreenProvider, useIsFullScreen } from "@/components/full-screen"
import {
  HeaderControls,
  LocaleSwitch,
  LogoutButton,
  ThemeButton,
} from "@/components/header-controls"
import { DeletionBanner } from "@/features/account"
import { AuthGate, AuthProvider, useAuth } from "@/features/auth"
import { cn } from "@/lib/utils"

const nav = [
  { href: "/learning", label: "topics", icon: IconStack2 },
  { href: "/learning/path", label: "pathTab", icon: IconChalkboardTeacher },
  { href: "/learning/chat", label: "chatTab", icon: IconMessageDots },
  { href: "/learning/settings", label: "settings", icon: IconSettings },
] as const

/** Topic pages live at /learning/<id>, so Topics owns whatever no other tab claims. */
function isActive(href: string, pathname: string) {
  if (href !== "/learning") return pathname.startsWith(href)

  return !nav.slice(1).some((item) => pathname.startsWith(item.href))
}

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join("")
}

function Logo() {
  return (
    <Link href="/learning" className="flex items-center gap-2.5">
      <span className="bg-primary text-primary-foreground flex size-7 items-center justify-center rounded-md">
        <IconSchool size={15} />
      </span>
      <span className="text-[15px] font-semibold">learning</span>
    </Link>
  )
}

function Sidebar() {
  const pathname = usePathname()
  const t = useTranslations("learning")
  const { user } = useAuth()
  const name = user?.first_name ?? ""

  return (
    <aside className="border-border bg-background sticky top-0 hidden h-dvh w-[272px] shrink-0 flex-col gap-4 border-r px-4 py-5 lg:flex">
      <div className="px-2 py-1">
        <Logo />
      </div>
      <nav aria-label={t("title")} className="flex flex-col gap-0.5">
        {nav.map(({ href, label, icon: Icon }) => {
          const active = isActive(href, pathname)

          return (
            <Link
              key={href}
              href={href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex h-9 items-center gap-2.5 rounded-md px-3 text-sm font-medium transition-colors",
                active
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
              )}
            >
              <Icon size={16} />
              {t(label)}
            </Link>
          )
        })}
      </nav>
      <div className="mt-auto flex flex-col gap-3">
        <LocaleSwitch />
        <div className="flex items-center gap-2.5 px-2 py-1">
          <span className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold">
            {initials(name)}
          </span>
          <span className="min-w-0 flex-1 truncate text-sm font-medium">
            {name}
          </span>
          <ThemeButton className="size-8 border-transparent bg-transparent" />
          <LogoutButton className="size-8 border-transparent bg-transparent" />
        </div>
      </div>
    </aside>
  )
}

function Frame({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const t = useTranslations("learning")
  const fullScreen = useIsFullScreen()

  return (
    <div className="lg:flex">
      <Sidebar />
      <div className="min-w-0 flex-1">
        {!fullScreen && (
          <header className="bg-background border-border sticky top-0 z-20 border-b lg:hidden">
            <div className="mx-auto flex w-full max-w-md items-center justify-between px-4 py-3">
              <Logo />
              <HeaderControls />
            </div>
          </header>
        )}
        <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 px-4 pt-4 pb-44 lg:max-w-[1168px] lg:px-8 lg:pt-6 lg:pb-16">
          <DeletionBanner />
          {children}
        </main>
      </div>
      {!fullScreen && (
        <nav
          aria-label={t("title")}
          className="border-border bg-background fixed inset-x-0 bottom-0 z-20 border-t lg:hidden"
        >
          <div className="mx-auto flex max-w-md items-stretch px-2 pt-1.5 pb-[max(env(safe-area-inset-bottom),0.875rem)]">
            {nav.map(({ href, label, icon: Icon }) => {
              const active = isActive(href, pathname)

              return (
                <Link
                  key={href}
                  href={href}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "flex flex-1 flex-col items-center gap-1 py-1 text-[10px]",
                    active
                      ? "text-foreground font-semibold"
                      : "text-faint font-medium",
                  )}
                >
                  <Icon size={18} />
                  {t(label)}
                </Link>
              )
            })}
          </div>
        </nav>
      )}
    </div>
  )
}

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <AuthGate>
        <FullScreenProvider>
          <Frame>{children}</Frame>
        </FullScreenProvider>
      </AuthGate>
    </AuthProvider>
  )
}
