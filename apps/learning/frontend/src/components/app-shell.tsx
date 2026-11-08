"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"
import { IconBook2, IconMessageCircle, IconPlus, IconSettings } from "@tabler/icons-react"
import { AddActionProvider, useAddAction } from "@/components/add-action"
import { FullScreenProvider, useIsFullScreen } from "@/components/full-screen"
import { HeaderControls } from "@/components/header-controls"
import { DeletionBanner } from "@/features/account"
import { AuthGate, AuthProvider } from "@/features/auth"

function Frame({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const t = useTranslations("learning")
  const { trigger, enabled } = useAddAction()
  const fullScreen = useIsFullScreen()
  return (
    <div className="lg:flex">
      <aside className="border-border bg-background sticky top-0 hidden h-dvh w-56 shrink-0 flex-col border-r px-3.5 py-5 lg:flex">
        <Link href="/learning" className="px-2 pb-6 text-[15px] font-bold">learning</Link>
        <Link href="/learning" className="text-muted-foreground flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm"><IconBook2 size={18} />{t("title")}</Link>
        <Link href="/learning/chat" className="text-muted-foreground flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm"><IconMessageCircle size={18} />{t("chatTab")}</Link>
        <Link href="/learning/settings" className="text-muted-foreground flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm"><IconSettings size={18} />{t("settings")}</Link>
        <div className="mt-auto flex justify-center"><HeaderControls /></div>
      </aside>
      <div className="min-w-0 flex-1">
        {!fullScreen && (
          <header className="bg-background border-border sticky top-0 z-20 border-b lg:hidden">
            <div className="mx-auto flex w-full max-w-md items-center justify-between px-5 py-4"><Link href="/learning" className="text-[15px] font-semibold">learning</Link><HeaderControls /></div>
          </header>
        )}
        <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 px-[18px] pt-[18px] pb-44 lg:max-w-[1040px] lg:px-6 lg:pt-6 lg:pb-16"><DeletionBanner />{children}</main>
      </div>
      {!fullScreen && (
      <nav aria-label={t("title")} className="border-border fixed inset-x-0 bottom-0 z-20 border-t bg-background lg:hidden">
        <div className="mx-auto flex max-w-md items-center justify-around px-4 py-4 pb-[max(env(safe-area-inset-bottom),1.375rem)]">
          <Link href="/learning" aria-current={pathname === "/learning" ? "page" : undefined} className="text-primary flex flex-col items-center gap-1 text-xs"><IconBook2 size={22} />{t("title")}</Link>
          <Link href="/learning/chat" aria-current={pathname === "/learning/chat" ? "page" : undefined} className="text-primary flex flex-col items-center gap-1 text-xs"><IconMessageCircle size={22} />{t("chatTab")}</Link>
          <button type="button" onClick={trigger} disabled={!enabled} aria-label={t("newTopic")} className="bg-primary text-primary-foreground flex size-11 items-center justify-center rounded-full disabled:opacity-40"><IconPlus size={22} /></button>
          <Link href="/learning/settings" aria-current={pathname === "/learning/settings" ? "page" : undefined} className="text-primary flex flex-col items-center gap-1 text-xs"><IconSettings size={22} />{t("settings")}</Link>
        </div>
      </nav>
      )}
    </div>
  )
}

export function AppShell({ children }: { children: React.ReactNode }) {
  return <AuthProvider><AuthGate><AddActionProvider><FullScreenProvider><Frame>{children}</Frame></FullScreenProvider></AddActionProvider></AuthGate></AuthProvider>
}
