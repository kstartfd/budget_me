"use client"

import { useAuth } from "@/features/auth"

/**
 * The two login identities of the current account. Pure derivation over
 * useAuth().user — the session user is the single source of truth (GET /me
 * today, patched optimistically by the link mutations), so there is no fetch of
 * its own. `identityCount` backs the hard rule: never offer to remove the last
 * remaining identity.
 */
export interface AccountIdentities {
  email: string | null
  telegramId: number | null
  username: string | null
  hasEmail: boolean
  hasTelegram: boolean
  identityCount: number
}

export function useAccount(): AccountIdentities {
  const { user } = useAuth()
  const email = user?.email ?? null
  const telegramId = user?.telegram_id ?? null
  const username = user?.username ?? null
  const hasEmail = email !== null
  const hasTelegram = telegramId !== null
  return {
    email,
    telegramId,
    username,
    hasEmail,
    hasTelegram,
    identityCount: (hasEmail ? 1 : 0) + (hasTelegram ? 1 : 0),
  }
}
