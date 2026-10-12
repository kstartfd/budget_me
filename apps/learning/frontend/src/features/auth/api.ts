import {
  authResponseSchema,
  userSchema,
  type AuthResponse,
  type User,
} from "./schemas"

/**
 * Real HTTP fetchers against the Go backend (same-origin /api/v1, proxied by
 * Next rewrites). Every response uses the standard envelope
 * `{code, message, data}` — non-zero code becomes an ApiError with the
 * (already localized) message.
 */

export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: number,
    readonly status: number,
  ) {
    super(message)
  }
}

async function call<T>(
  path: string,
  init: RequestInit,
  parse: (data: unknown) => T,
): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init.headers },
  })
  const body = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!body || body.code !== 0) {
    throw new ApiError(body?.message ?? res.statusText, body?.code ?? -1, res.status)
  }
  return parse(body.data)
}

export function telegramAuth(initData: string): Promise<AuthResponse> {
  return call(
    "/auth/telegram",
    { method: "POST", body: JSON.stringify({ init_data: initData }) },
    (d) => authResponseSchema.parse(d),
  )
}

export function login(input: {
  email: string
  password: string
}): Promise<AuthResponse> {
  return call(
    "/auth/login",
    { method: "POST", body: JSON.stringify(input) },
    (d) => authResponseSchema.parse(d),
  )
}

/** Validates a stored token and returns the current user. */
export function me(token: string): Promise<User> {
  return call(
    "/me",
    { method: "GET", headers: { Authorization: `Bearer ${token}` } },
    (d) => userSchema.parse(d),
  )
}

/**
 * Links an email/password login path onto the authenticated account.
 * Returns the updated user (now carrying `email`); a 409 (envelope code 40001,
 * "email already registered") means the email belongs to another account.
 */
export function linkEmail(
  token: string,
  input: { email: string; password: string },
): Promise<User> {
  return call(
    "/auth/link/email",
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify(input),
    },
    (d) => userSchema.parse(d),
  )
}
