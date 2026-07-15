"use client"

import { useSyncExternalStore } from "react"

/**
 * `true` on wide (lg+) viewports, tracking Tailwind's `lg` breakpoint
 * (1024px). Hydration-safe: the server snapshot is `false` (mobile-first), so
 * markup matches until the client subscribes. Used to switch a component's
 * whole presentation by breakpoint (e.g. bottom sheet vs. centered dialog)
 * where a CSS `lg:` toggle can't — both variants would otherwise mount.
 */

const QUERY = "(min-width: 1024px)"

export function useIsDesktop(): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mq = window.matchMedia(QUERY)
      mq.addEventListener("change", onChange)
      return () => mq.removeEventListener("change", onChange)
    },
    () => window.matchMedia(QUERY).matches,
    () => false,
  )
}
