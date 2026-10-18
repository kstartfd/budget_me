"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react"

/**
 * Nested screens (a topic, an entry) take the whole phone: the brand header and
 * the tab bar step aside and the screen draws its own back bar. A screen claims
 * that for as long as it is mounted; on desktop the sidebar stays either way.
 */

interface FullScreenContextValue {
  active: boolean
  claim: () => () => void
}

const FullScreenContext = createContext<FullScreenContextValue | null>(null)

export function FullScreenProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [claims, setClaims] = useState(0)

  const claim = useCallback(() => {
    setClaims((n) => n + 1)

    return () => setClaims((n) => n - 1)
  }, [])

  const value = useMemo(
    () => ({ active: claims > 0, claim }),
    [claims, claim],
  )

  return (
    <FullScreenContext.Provider value={value}>
      {children}
    </FullScreenContext.Provider>
  )
}

/** Hide the phone chrome for as long as this screen is mounted. */
export function useClaimFullScreen() {
  const claim = useContext(FullScreenContext)?.claim

  useEffect(() => {
    if (!claim) return

    return claim()
  }, [claim])
}

export function useIsFullScreen(): boolean {
  return useContext(FullScreenContext)?.active ?? false
}
