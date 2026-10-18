"use client"

import { useAuth } from "@/features/auth"

/**
 * The login identities of the current account, derived from the session user
 * (GET /me, patched by the link mutations) — no fetch of its own.
 */
export interface AccountIdentities {
  email: string | null
  telegramId: number | null
  username: string | null
  hasEmail: boolean
  hasTelegram: boolean
}

export function useAccount(): AccountIdentities {
  const { user } = useAuth()
  const email = user?.email ?? null
  const telegramId = user?.telegram_id ?? null

  return {
    email,
    telegramId,
    username: user?.username ?? null,
    hasEmail: email !== null,
    hasTelegram: telegramId !== null,
  }
}
