"use client"

import type { ComponentType } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"

import { useAddAction } from "@/components/add-action"
import { PlusIcon } from "@/components/app-icons"
import { useSpace } from "@/components/spaces"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { cn } from "@/lib/utils"

/**
 * Design bottom nav: four tabs around a raised center ＋. The four come from
 * the active space (see components/spaces.tsx) — «Деньги» keeps the original
 * Обзор / Траты / Графики / AI, «Жизнь» shows День / Списки / English / Ещё.
 *
 * The ＋ is contextual. In «Деньги» it is the add-expense sheet it has always
 * been; in «Жизнь» it belongs to the open screen, which claims it through
 * useProvideAddAction — so it stays disabled on screens that are still stubs
 * rather than opening someone else's sheet.
 */
export function BottomNav() {
  const pathname = usePathname()
  const t = useTranslations("nav")
  const tAdd = useTranslations("add")
  const { openAdd } = useBudgetUI()
  const space = useSpace()
  const { trigger, enabled } = useAddAction()

  const left = space.tabs.slice(0, 2)
  const right = space.tabs.slice(2)

  const money = space.id === "money"
  const addLabel = tAdd(
    space.tabs.find((tab) => tab.href === pathname)?.addKey ?? space.addKey,
  )
  const addEnabled = money || enabled

  return (
    <nav
      aria-label={t("main")}
      className="border-border fixed inset-x-0 bottom-0 z-20 border-t lg:hidden"
      style={{
        background: "var(--nav-bg)",
        backdropFilter: "blur(12px)",
        WebkitBackdropFilter: "blur(12px)",
      }}
    >
      <div className="mx-auto flex w-full max-w-md items-end justify-around px-1.5 pt-[9px] pb-[max(env(safe-area-inset-bottom),1.375rem)]">
        {left.map((tab) => (
          <NavTab
            key={tab.href}
            href={tab.href}
            icon={tab.icon}
            label={t(tab.labelKey)}
            active={pathname === tab.href}
          />
        ))}

        <button
          type="button"
          onClick={money ? openAdd : trigger}
          disabled={!addEnabled}
          aria-label={addLabel}
          className="flex flex-1 flex-col items-center gap-1 py-0.5 disabled:opacity-40"
        >
          <span className="bg-primary text-primary-foreground -mt-3 flex size-[46px] items-center justify-center rounded-full shadow-[0_10px_22px_-8px_var(--primary-glow)]">
            <PlusIcon />
          </span>
          <span className="text-primary text-[10px] font-semibold">
            {addLabel}
          </span>
        </button>

        {right.map((tab) => (
          <NavTab
            key={tab.href}
            href={tab.href}
            icon={tab.icon}
            label={t(tab.labelKey)}
            active={pathname === tab.href}
          />
        ))}
      </div>
    </nav>
  )
}

function NavTab({
  href,
  icon: Icon,
  label,
  active,
}: {
  href: string
  icon: ComponentType<{ size?: number }>
  label: string
  active: boolean
}) {
  return (
    <Link
      href={href}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex flex-1 flex-col items-center gap-1 py-0.5",
        active ? "text-primary" : "text-faint",
      )}
    >
      <Icon />
      <span className="text-[10px] font-medium">{label}</span>
    </Link>
  )
}
