"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"

import { useAddAction } from "@/components/add-action"
import { PlusIcon, WalletLogoIcon } from "@/components/app-icons"
import { HeaderControls } from "@/components/header-controls"
import { SPACES, tabMatches, useSpace } from "@/components/spaces"
import { useAuth } from "@/features/auth"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { cn } from "@/lib/utils"

/**
 * Desktop left sidebar (design WEB FRAME 1): brand, the signed-in user, the
 * nav, a prominent add button, and the shared locale/theme controls pinned to
 * the bottom. Shown only on wide viewports (lg+).
 *
 * Unlike the phone there is no space switcher here: the sidebar has the room
 * to show both groups — ДЕНЬГИ and ЖИЗНЬ — at once, which is faster than a
 * toggle. The add button still follows the open screen, exactly as the phone's
 * center ＋ does. The gear lives in HeaderControls, so «Ещё» (a bottom-bar-only
 * tab) is left out here.
 */
export function DesktopNav() {
  const pathname = usePathname()
  const t = useTranslations("nav")
  const tSpaces = useTranslations("spaces")
  const tAdd = useTranslations("add")
  const { openAdd } = useBudgetUI()
  const { trigger, enabled } = useAddAction()
  const { user } = useAuth()
  const space = useSpace()

  const money = space.id === "money"
  const addLabel = tAdd(
    space.tabs.find((tab) => tabMatches(tab, pathname))?.addKey ?? space.addKey,
  )
  const addEnabled = money || enabled

  const displayName = user?.first_name || user?.username || ""
  const handle = user?.username ? `@${user.username}` : null
  const initial = (displayName || "?").charAt(0).toUpperCase()

  return (
    <aside className="border-border bg-background sticky top-0 hidden h-dvh w-56 shrink-0 flex-col border-r px-3.5 py-5 lg:flex">
      <Link
        href="/"
        className="flex items-center gap-2.5 px-2 pb-4 text-[15px] font-bold tracking-[-0.01em]"
      >
        <span className="bg-primary text-primary-foreground flex size-8 items-center justify-center rounded-[10px]">
          <WalletLogoIcon />
        </span>
        budget_me
      </Link>

      {user && (
        <div className="border-border mb-2 flex items-center gap-2.5 border-b px-2 pb-3">
          <span className="bg-secondary text-muted-foreground flex size-[30px] shrink-0 items-center justify-center rounded-full text-xs font-semibold">
            {initial}
          </span>
          <div className="min-w-0 leading-tight">
            {displayName && (
              <div className="text-foreground truncate text-[12.5px] font-semibold">
                {displayName}
              </div>
            )}
            {handle && (
              <div className="text-faint truncate text-[11px]">{handle}</div>
            )}
          </div>
        </div>
      )}

      <div className="flex flex-col gap-3.5">
        {SPACES.map((group) => (
          <nav key={group.id} aria-label={tSpaces(group.labelKey)}>
            <div className="text-faint px-3 pb-1.5 font-mono text-[10px] font-bold tracking-[0.09em] uppercase">
              {tSpaces(group.labelKey)}
            </div>
            <div className="flex flex-col gap-1.5">
              {group.tabs
                .filter((tab) => !tab.bottomOnly)
                .map((tab) => {
                  const { href, icon: Icon, labelKey } = tab
                  const active = tabMatches(tab, pathname)
                  return (
                    <Link
                      key={href}
                      href={href}
                      aria-current={active ? "page" : undefined}
                      className={cn(
                        "focus-visible:ring-ring flex items-center gap-3 rounded-xl px-3 py-2.5 text-[13.5px] outline-none focus-visible:ring-2",
                        active
                          ? "bg-[color:var(--primary-wash)] text-primary font-semibold"
                          : "text-muted-foreground hover:text-foreground hover:bg-secondary/60 font-medium",
                      )}
                    >
                      <Icon size={18} />
                      {t(labelKey)}
                    </Link>
                  )
                })}
            </div>
          </nav>
        ))}
      </div>

      <button
        type="button"
        onClick={money ? openAdd : trigger}
        disabled={!addEnabled}
        className="bg-primary text-primary-foreground focus-visible:ring-ring mt-4 flex h-11 items-center justify-center gap-2 rounded-xl text-sm font-bold shadow-[0_10px_22px_-8px_var(--primary-glow)] outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:opacity-40"
      >
        <PlusIcon size={18} />
        {addLabel}
      </button>

      <div className="mt-auto flex justify-center pt-4">
        <HeaderControls />
      </div>
    </aside>
  )
}
