"use client"

import { useState } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"
import {
  IconBook2,
  IconBolt,
  IconChartLine,
  IconClipboardCheck,
  IconLayoutDashboard,
  IconSettings,
  IconMenu2,
  IconMessageDots,
  IconRoute,
} from "@tabler/icons-react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { FullScreenProvider, useIsFullScreen } from "@/components/full-screen"
import {
  LocaleSwitch,
  LogoutButton,
  ThemeButton,
} from "@/components/header-controls"
import { DeletionBanner } from "@/features/account"
import { AuthGate, AuthProvider, useAuth } from "@/features/auth"
import { LanguageSwitcher } from "@/features/home"
import { useActiveTeacher } from "@/features/home/setup"
import { cn } from "@/lib/utils"

const nav = [
  { href: "/learning/overview", label: "overviewTab", icon: IconLayoutDashboard },
  { href: "/learning/path", label: "pathTab", icon: IconRoute },
  { href: "/learning/knowledge", label: "knowledgeTab", icon: IconBook2 },
  { href: "/learning/practice", label: "practiceTab", icon: IconBolt },
  { href: "/learning/assessment", label: "assessmentTab", icon: IconClipboardCheck },
  { href: "/learning/progress", label: "progressTab", icon: IconChartLine },
  { href: "/learning/chat", label: "chatTab", icon: IconMessageDots },
  { href: "/learning/settings", label: "settings", icon: IconSettings },
] as const

/** The pages that belong to the Overview tab besides its own: setting up and managing languages. */
const overviewPages = ["/learning/overview", "/learning/start"]

function ownedBy(href: string, pathname: string) {
  if (href === "/learning/overview")
    return overviewPages.some((page) => pathname.startsWith(page))
  if (href === "/learning/settings")
    return pathname.startsWith(href) || pathname === "/learning/languages" || pathname === "/learning/tokens"
  if (href === "/learning/knowledge")
    return pathname.startsWith("/learning/knowledge") || pathname.startsWith("/learning/path/rules")
  if (href === "/learning/path")
    return pathname === href
  if (href === "/learning/practice")
    return pathname === "/learning" || pathname.startsWith(href) || (
      /^\/learning\/[^/]+$/.test(pathname) &&
      !nav.some((item) => item.href === pathname) &&
      !overviewPages.includes(pathname) &&
      pathname !== "/learning/tokens"
    )
  return pathname.startsWith(href)
}

function isActive(href: string, pathname: string) {
  return ownedBy(href, pathname)
}

function Logo() {
  return (
    <Link href="/learning/overview" className="flex items-center gap-2 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
      <span className="bg-primary text-primary-foreground flex size-7 items-center justify-center rounded-md text-sm font-semibold">
        l
      </span>
      <span className="text-[14px] font-semibold tracking-[-0.01em]">learning</span>
    </Link>
  )
}

function Navigation({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname()
  const t = useTranslations("learning")
  const { active } = useActiveTeacher()

  return (
    <nav aria-label={t("navigation")} className="flex flex-col gap-0.5">
      {nav.map(({ href, label, icon: Icon }) => {
        const selected = isActive(href, pathname)
        const destination = active && ["/learning/path", "/learning/knowledge", "/learning/practice", "/learning/progress"].includes(href)
          ? `${href}?language=${encodeURIComponent(active.language)}`
          : href

        return (
          <Link
            key={href}
            href={destination}
            onClick={onNavigate}
            aria-current={selected ? "page" : undefined}
            className={cn(
              "flex min-h-9 items-center gap-2.5 rounded-md px-3 text-[12px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
              selected
                ? "border border-[#cececb] bg-[#f0f0ee] text-sidebar-accent-foreground"
                : "text-[#686864] hover:bg-[#f5f5f3] hover:text-sidebar-accent-foreground",
            )}
          >
            <Icon size={15} stroke={1.8} aria-hidden="true" />
            {t(label)}
          </Link>
        )
      })}
    </nav>
  )
}

function Sidebar() {
  const { user } = useAuth()
  const workspace = useTranslations("workspace")
  const name = user?.first_name ?? ""

  return (
    <aside className="border-sidebar-border bg-sidebar text-sidebar-foreground sticky top-0 hidden h-dvh w-[232px] shrink-0 flex-col border-r px-4 py-5 md:flex">
      <div className="mb-5 px-2">
        <Logo />
      </div>
      <LanguageSwitcher className="mb-3 min-h-[50px] text-[12px] [&_.language-switcher-detail]:text-[10px]" />
      <Navigation />
      <Link href="/learning/settings" className="border-sidebar-border hover:bg-sidebar-accent/60 mt-auto flex flex-col border-t pt-3 text-[12px] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
          <span className="truncate font-medium">{name}</span>
          <span className="text-muted-foreground text-[11px]">{workspace("accountSettings")}</span>
      </Link>
    </aside>
  )
}

function Frame({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const t = useTranslations("learning")
  const fullScreen = useIsFullScreen()
  const [menuOpen, setMenuOpen] = useState(false)
  const { active } = useActiveTeacher()
  const mobileNav = nav.filter((item) => ["/learning/overview", "/learning/path", "/learning/practice", "/learning/assessment"].includes(item.href))

  return (
    <div className="group md:flex">
      <Sidebar />
      <div className="min-w-0 flex-1">
        {!fullScreen && (
          // 61px tall: what sticks under it (the open topic's row) sits at that offset.
          <header className="bg-background border-border sticky top-0 z-20 border-b md:hidden">
            <div className="mx-auto flex w-full max-w-md items-center justify-between px-4 py-3">
              <div className="flex items-center gap-2">
                <button type="button" aria-label={t("openNavigation")} onClick={() => setMenuOpen(true)} className="hover:bg-muted flex size-9 items-center justify-center rounded-lg focus-visible:outline-2 focus-visible:outline-ring">
                  <IconMenu2 size={20} aria-hidden="true" />
                </button>
                <Logo />
              </div>
              <ThemeButton />
            </div>
          </header>
        )}
        <Dialog open={menuOpen} onOpenChange={setMenuOpen}>
          <DialogContent className="top-0 left-0 flex h-dvh max-h-dvh w-[min(88vw,340px)] max-w-none translate-x-0 translate-y-0 flex-col gap-4 rounded-none border-l-0 p-4 sm:max-w-none">
            <DialogHeader className="mt-3 text-left"><DialogTitle><Logo /></DialogTitle></DialogHeader>
            <div className="mt-2"><LanguageSwitcher /></div>
            <Navigation onNavigate={() => setMenuOpen(false)} />
            <div className="mt-auto flex items-center gap-2"><LocaleSwitch /><LogoutButton /></div>
          </DialogContent>
        </Dialog>
        <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 px-4 pt-4 pb-44 md:max-w-[1168px] md:px-8 md:pt-6 md:pb-16">
          <DeletionBanner />
          {children}
        </main>
      </div>
      {!fullScreen && (
        <nav
          aria-label={t("title")}
          className="border-border bg-background fixed inset-x-0 bottom-0 z-20 border-t md:hidden"
        >
          <div className="mx-auto flex max-w-md items-stretch px-2 pt-1.5 pb-[max(env(safe-area-inset-bottom),0.875rem)]">
            {mobileNav.map(({ href, label, icon: Icon }) => {
              const selected = isActive(href, pathname)
              const destination = active && ["/learning/path", "/learning/practice"].includes(href)
                ? `${href}?language=${encodeURIComponent(active.language)}` : href

              return (
                <Link
                  key={href}
                  href={destination}
                  aria-current={selected ? "page" : undefined}
                  className={cn(
                    "flex flex-1 flex-col items-center gap-1 rounded-md py-1 text-[10px] focus-visible:outline-2 focus-visible:outline-ring",
                    selected
                      ? "text-foreground font-semibold"
                      : "text-faint font-medium",
                  )}
                >
                  <Icon size={18} aria-hidden="true" />
                  {t(label)}
                </Link>
              )
            })}
            <button type="button" onClick={() => setMenuOpen(true)} aria-label={t("moreNavigation")} className="text-faint flex flex-1 flex-col items-center gap-1 rounded-md py-1 text-[10px] font-medium focus-visible:outline-2 focus-visible:outline-ring">
              <IconMenu2 size={18} aria-hidden="true" />{t("more")}
            </button>
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
