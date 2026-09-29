"use client"

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react"

/**
 * The nav's center ＋ belongs to whichever «Жизнь» screen is open: on /day it
 * adds a task, on /lists a thing, on /learning a topic or entry. Screens claim it while
 * they are mounted instead of the shell importing every feature's sheet.
 *
 * The handler itself lives in a ref, so a screen that passes an unmemoized
 * callback re-registers without re-rendering the bar (a plain state slot would
 * loop). Only the boolean — is anything claiming the button — is state.
 */

interface AddActionContextValue {
  register: (fn: () => void) => () => void
  trigger: () => void
  enabled: boolean
}

const AddActionContext = createContext<AddActionContextValue | null>(null)

export function AddActionProvider({ children }: { children: React.ReactNode }) {
  const handler = useRef<(() => void) | null>(null)
  const [enabled, setEnabled] = useState(false)

  const register = useCallback((fn: () => void) => {
    handler.current = fn
    setEnabled(true)

    return () => {
      handler.current = null
      setEnabled(false)
    }
  }, [])

  const trigger = useCallback(() => handler.current?.(), [])

  return (
    <AddActionContext.Provider value={{ register, trigger, enabled }}>
      {children}
    </AddActionContext.Provider>
  )
}

/** Claim the nav's ＋ for as long as this screen is mounted. */
export function useProvideAddAction(fn: () => void) {
  const ctx = useContext(AddActionContext)
  const register = ctx?.register

  useEffect(() => {
    if (!register) return

    return register(fn)
  }, [register, fn])
}

export function useAddAction(): Omit<AddActionContextValue, "register"> {
  const ctx = useContext(AddActionContext)
  if (!ctx) throw new Error("useAddAction must be used inside AddActionProvider")

  return { trigger: ctx.trigger, enabled: ctx.enabled }
}
