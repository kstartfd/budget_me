"use client"

import { createContext, useContext, useState, type ComponentType } from "react"
import { usePathname, useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import {
  DotsIcon,
  HomeIcon,
  ListIcon,
  PieIcon,
  SparklesIcon,
  StackIcon,
} from "@/components/app-icons"
import { cn } from "@/lib/utils"

/**
 * The app is split into two SPACES — «Деньги» (the original budget screens)
 * and «Жизнь» (lists). One Mini App, one session, one
 * shell; the bottom bar simply swaps its four tabs, so the money screens keep
 * the navigation they have always had.
 *
 * Phone: the SpaceSwitcher in the header picks the space, and the bottom bar
 * follows. Desktop: no switcher at all — the sidebar shows both groups at
 * once (see DesktopNav), which is the whole point of having the room.
 */

export type SpaceId = "money" | "life"

export interface NavTab {
  href: string
  icon: ComponentType<{ size?: number }>
  /** Key under the `nav` message namespace. */
  labelKey: string
  /** Key under `add` for the center button's label while this tab is open. */
  addKey?: string
  /** Bottom bar only — the sidebar reaches it another way (settings: the gear). */
  bottomOnly?: boolean
}

export interface Space {
  id: SpaceId
  /** Key under the `spaces` message namespace. */
  labelKey: string
  /** Exactly four — the bottom bar hangs two on each side of the ＋ button. */
  tabs: NavTab[]
  /** Fallback label key for ＋ on routes that declare none. */
  addKey: string
}

export const SPACES: Space[] = [
  {
    id: "money",
    labelKey: "money",
    addKey: "fab",
    tabs: [
      { href: "/", icon: HomeIcon, labelKey: "overview" },
      { href: "/transactions", icon: ListIcon, labelKey: "transactions" },
      { href: "/analytics", icon: PieIcon, labelKey: "analytics" },
      { href: "/ai", icon: SparklesIcon, labelKey: "ai" },
    ],
  },
  {
    id: "life",
    labelKey: "life",
    addKey: "item",
    tabs: [
      { href: "/lists", icon: StackIcon, labelKey: "lists", addKey: "item" },
      { href: "/settings", icon: DotsIcon, labelKey: "more", bottomOnly: true },
    ],
  },
]

/**
 * Whether `pathname` belongs to `tab` — the tab's own route, or one nested
 * under it (`/lists` also claims `/lists/abc123`). A prefix match, not
 * `startsWith`, so `/lists` never claims a sibling like `/lists-export`.
 */
export function tabMatches(tab: NavTab, pathname: string): boolean {
  return pathname === tab.href || pathname.startsWith(`${tab.href}/`)
}

/**
 * Which space a route belongs to — null for the shared screens (settings,
 * categories), which belong to neither and must not yank the bar around.
 */
export function spaceOfPath(pathname: string): SpaceId | null {
  for (const space of SPACES) {
    if (space.tabs.some((tab) => !tab.bottomOnly && tabMatches(tab, pathname))) {
      return space.id
    }
  }

  return null
}

const SpaceContext = createContext<Space | null>(null)

export function SpaceProvider({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const fromPath = spaceOfPath(pathname)
  // Shared routes keep whichever space the user came from, so opening Settings
  // from «Жизнь» does not flip the bar back to «Деньги». The route wins when it
  // names a space; the remembered value only covers the routes that name none.
  // Adjusted during render (React's "adjust state when props change") rather
  // than in an effect, so the bar never paints one frame with the old space.
  const [remembered, setRemembered] = useState<SpaceId>(fromPath ?? "money")
  const activeId = fromPath ?? remembered

  if (fromPath && fromPath !== remembered) setRemembered(fromPath)

  const space = SPACES.find((s) => s.id === activeId) ?? SPACES[0]

  return <SpaceContext.Provider value={space}>{children}</SpaceContext.Provider>
}

export function useSpace(): Space {
  const space = useContext(SpaceContext)
  if (!space) throw new Error("useSpace must be used inside SpaceProvider")

  return space
}

/**
 * Two-pill segment in the mobile header. Switching navigates to the space's
 * first tab — a space is a place, not a filter, so it has to land somewhere.
 */
export function SpaceSwitcher() {
  const t = useTranslations("spaces")
  const router = useRouter()
  const active = useSpace()

  return (
    <div
      role="tablist"
      aria-label={t("aria")}
      className="bg-secondary flex gap-1 rounded-full p-1"
    >
      {SPACES.map((space) => {
        const on = space.id === active.id
        return (
          <button
            key={space.id}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => router.push(space.tabs[0].href)}
            className={cn(
              "flex-1 rounded-full py-[7px] text-[12.5px] font-semibold transition-colors",
              on ? "bg-card text-foreground shadow-[var(--shadow-card)]" : "text-faint",
            )}
          >
            {t(space.labelKey)}
          </button>
        )
      })}
    </div>
  )
}
