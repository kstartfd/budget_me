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
 * Each hook posts to a protected endpoint, then merges the backend-confirmed
 * result into the session user via `patchUser`, so the screen flips at once
 * with no separate query to invalidate. Failures surface as auth's `ApiError`
 * (409: the email belongs to another account).
 */
export function useLinkEmail() {
  const { patchUser } = useAuth()

  return useMutation({
    mutationFn: (input: LinkEmailFormValues) => linkEmail(requireToken(), input),
    onSuccess: (user) => patchUser(user),
  })
}

/** Nothing is erased now: the account keeps working until the deadline. */
export function useRequestAccountDeletion() {
  const { patchUser } = useAuth()

  return useMutation({
    mutationFn: () => requestAccountDeletion(requireToken()),
    onSuccess: (deleteAt) => patchUser({ delete_at: deleteAt ?? undefined }),
  })
}

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
