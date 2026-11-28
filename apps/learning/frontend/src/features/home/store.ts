"use client"

import { useCallback, useMemo, useSyncExternalStore } from "react"

/**
 * A small value kept in the browser, per learner. It is where what the backend
 * does not keep yet lives (the answers of the setup wizard, how far the setup
 * of a language has got), and where the learner's own choice of the language on
 * screen lives. Everything here is read through one interface so that moving it
 * to the server later changes this file and nothing that uses it.
 */

const listeners = new Set<() => void>()

function subscribe(onChange: () => void) {
  listeners.add(onChange)
  window.addEventListener("storage", onChange)

  return () => {
    listeners.delete(onChange)
    window.removeEventListener("storage", onChange)
  }
}

function read(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function write(key: string, value: string | null) {
  try {
    if (value === null) window.localStorage.removeItem(key)
    else window.localStorage.setItem(key, value)
  } catch {
    // A webview with storage turned off keeps the value for this visit only.
  }

  listeners.forEach((listener) => listener())
}

/**
 * The value under `key`, parsed by `parse` (which returns null for anything it
 * does not recognise, so a stale or hand-edited value is just absent), and a
 * setter. A null key is "not signed in yet": nothing is read or written.
 */
export function useStored<T>(
  key: string | null,
  parse: (raw: unknown) => T | null,
): [T | null, (value: T | null) => void] {
  const raw = useSyncExternalStore(
    subscribe,
    () => (key ? read(key) : null),
    () => null,
  )

  const value = useMemo(() => {
    if (!raw) return null

    try {
      return parse(JSON.parse(raw))
    } catch {
      return null
    }
  }, [raw, parse])

  const set = useCallback(
    (next: T | null) => {
      if (key) write(key, next === null ? null : JSON.stringify(next))
    },
    [key],
  )

  return [value, set]
}
