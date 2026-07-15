"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react"

import { useTelegram } from "@/features/telegram/telegram-context"

import * as api from "./api"
import type { User } from "./schemas"

/**
 * Session state for the whole app. Boot order:
 *  1. a stored token is validated via GET /me (web session survives reloads),
 *  2. otherwise, inside Telegram, initData is exchanged automatically,
 *  3. otherwise the visitor is a guest — the AuthGate shows login (new
 *     accounts are created only inside Telegram; there is no web sign-up).
 *
 * The token is kept in localStorage: the Mini App webview has no safer
 * cross-restart storage, and the web fallback matches it for simplicity.
 */

const tokenKey = "budget_me.token.v1"

type AuthStatus = "booting" | "authed" | "guest"

interface AuthContextValue {
  status: AuthStatus
  user: User | null
  /** Non-null when automatic Telegram sign-in failed (shown with a retry). */
  bootError: string | null
  login: (email: string, password: string) => Promise<void>
  retryTelegram: () => void
  logout: () => void
  /**
   * Merge partial fields into the session user in place. Used by the «Аккаунт»
   * settings flow to reflect a freshly linked identity (email / telegram_id +
   * username) the moment the link succeeds.
   *
   * NOTE (mock): while the link runs on the mock layer this is optimistic-only —
   * the backend does not persist it yet, so a real GET /me on the next boot would
   * drop the merge. Once POST /settings/link-* lands it returns the updated user
   * and this stays the single place the session user is refreshed.
   */
  patchUser: (patch: Partial<User>) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

/** Reads the stored session token (or null). Exposed so other features can
 *  authorize their own backend requests without duplicating the storage key. */
export function readToken(): string | null {
  try {
    return window.localStorage.getItem(tokenKey)
  } catch {
    return null
  }
}

function writeToken(token: string | null): void {
  try {
    if (token === null) window.localStorage.removeItem(tokenKey)
    else window.localStorage.setItem(tokenKey, token)
  } catch {
    // Private mode: the session lives until the page closes.
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { webApp, ready } = useTelegram()
  const [status, setStatus] = useState<AuthStatus>("booting")
  const [user, setUser] = useState<User | null>(null)
  const [bootError, setBootError] = useState<string | null>(null)
  const booted = useRef(false)

  const finishAuthed = useCallback((token: string, u: User) => {
    writeToken(token)
    setUser(u)
    setBootError(null)
    setStatus("authed")
  }, [])

  const boot = useCallback(async () => {
    const stored = readToken()
    if (stored) {
      try {
        const u = await api.me(stored)
        setUser(u)
        setStatus("authed")
        return
      } catch (err) {
        // Expired/foreign token — drop it. Network failures keep the token
        // and fall through to the Telegram path / guest screen.
        if (err instanceof api.ApiError) writeToken(null)
      }
    }

    const initData = webApp?.initData ?? ""
    if (initData.length > 0) {
      try {
        const session = await api.telegramAuth(initData)
        finishAuthed(session.token, session.user)
        return
      } catch (err) {
        setBootError(err instanceof Error ? err.message : String(err))
      }
    }
    setStatus("guest")
  }, [webApp, finishAuthed])

  useEffect(() => {
    // Wait for the Telegram runtime to settle so initData is readable.
    if (!ready || booted.current) return
    booted.current = true
    void boot()
  }, [ready, boot])

  const value: AuthContextValue = {
    status,
    user,
    bootError,
    login: useCallback(
      async (email, password) => {
        const session = await api.login({ email, password })
        finishAuthed(session.token, session.user)
      },
      [finishAuthed],
    ),
    retryTelegram: useCallback(() => {
      setStatus("booting")
      setBootError(null)
      void boot()
    }, [boot]),
    logout: useCallback(() => {
      writeToken(null)
      setUser(null)
      setStatus("guest")
    }, []),
    patchUser: useCallback((patch: Partial<User>) => {
      setUser((u) => (u ? { ...u, ...patch } : u))
    }, []),
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider")
  return ctx
}
