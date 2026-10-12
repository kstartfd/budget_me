import { useSyncExternalStore } from "react"

const noop = () => () => {}

/**
 * `false` during SSR and the first client render, `true` after hydration.
 *
 * Uses `useSyncExternalStore` (server vs client snapshot) instead of
 * `useState`+`useEffect`, so it is hydration-safe and does not trip the
 * `react-hooks/set-state-in-effect` rule.
 */
export function useMounted(): boolean {
  return useSyncExternalStore(
    noop,
    () => true,
    () => false,
  )
}
