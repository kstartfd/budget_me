"use client"

import type { ComponentType } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"

import {
  HomeIcon,
  ListIcon,
  PieIcon,
  PlusIcon,
  SparklesIcon,
} from "@/components/app-icons"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { cn } from "@/lib/utils"

/**
 * Design bottom nav: four tabs (Overview / Spending · Charts / AI) around a
 * raised center "add" button. Limits moved into Settings (reached from the
 * header gear), and the add action now anchors the middle of the bar instead
 * of floating over it.
 */
export function BottomNav() {
  const pathname = usePathname()
  const t = useTranslations("nav")
  const tAdd = useTranslations("add")
  const { openAdd } = useBudgetUI()

  const left = [
    { href: "/", icon: HomeIcon, label: t("overview") },
    { href: "/transactions", icon: ListIcon, label: t("transactions") },
  ]
  const right = [
    { href: "/analytics", icon: PieIcon, label: t("analytics") },
    { href: "/ai", icon: SparklesIcon, label: t("ai") },
  ]

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
          <NavTab key={tab.href} {...tab} active={pathname === tab.href} />
        ))}

        <button
          type="button"
          onClick={openAdd}
          aria-label={tAdd("fab")}
          className="flex flex-1 flex-col items-center gap-1 py-0.5"
        >
          <span className="bg-primary text-primary-foreground -mt-3 flex size-[46px] items-center justify-center rounded-full shadow-[0_10px_22px_-8px_var(--primary-glow)]">
            <PlusIcon />
          </span>
          <span className="text-primary text-[10px] font-semibold">
            {tAdd("fab")}
          </span>
        </button>

        {right.map((tab) => (
          <NavTab key={tab.href} {...tab} active={pathname === tab.href} />
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
