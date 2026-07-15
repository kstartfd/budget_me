"use client"

import { useMutation } from "@tanstack/react-query"

import {
  ApiError,
  cancelAccountDeletion,
  linkEmail,
  readToken,
  requestAccountDeletion,
  useAuth,
} from "@/features/auth"

import type { LinkEmailFormValues } from "./schemas"

/**
 * Link a second login path onto the current account. Both hooks post to the
 * protected link endpoint, then merge the returned (full, backend-confirmed)
 * user into the session user via auth's `patchUser` — the session user is the
 * single source of truth GET /me populated, so the identity row flips to
 * "linked" at once with no separate me-query to invalidate.
 *
 * Errors surface as auth's `ApiError`: 409 for an identity already linked to
 * another account, 401 for invalid/expired Telegram initData. The calling
 * component maps the status to a toast and keeps its form open.
 */
export function useLinkEmail() {
  const { patchUser } = useAuth()

  return useMutation({
    mutationFn: (input: LinkEmailFormValues) => linkEmail(requireToken(), input),
    onSuccess: (user) => patchUser(user),
  })
}

/**
 * Ask for the account to be deleted. Nothing disappears now: the backend stamps
 * the request, the app keeps working, and a cron worker erases everything once
 * the grace period expires. The returned deadline lands on the session user, so
 * the banner offering to cancel appears straight away.
 */
export function useRequestAccountDeletion() {
  const { patchUser } = useAuth()

  return useMutation({
    mutationFn: () => requestAccountDeletion(requireToken()),
    onSuccess: (deleteAt) => patchUser({ delete_at: deleteAt ?? undefined }),
  })
}

/** Call the pending deletion off — the account (and its data) stay. */
export function useCancelAccountDeletion() {
  const { patchUser } = useAuth()

  return useMutation({
    mutationFn: () => cancelAccountDeletion(requireToken()),
    onSuccess: () => patchUser({ delete_at: undefined }),
  })
}

function requireToken(): string {
  const token = readToken()

  if (!token) throw new ApiError("no session token", -1, 401)

  return token
}
