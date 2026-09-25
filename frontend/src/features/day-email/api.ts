import { z } from "zod"

import { readToken } from "@/features/auth"

const preferenceSchema = z.object({
  email: z.string().nullable(),
  frequency: z.number().int().min(0).max(2),
  status: z.enum(["off", "pending", "active"]),
  configured: z.boolean(),
})

export type DayEmailPreference = z.infer<typeof preferenceSchema>

async function call(
  path: string,
  method: string,
  body?: object,
): Promise<DayEmailPreference> {
  const token = readToken()
  const res = await fetch(`/api/v1${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })
  const envelope = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!res.ok || !envelope || envelope.code !== 0) {
    throw new Error(envelope?.message ?? res.statusText)
  }
  return preferenceSchema.parse(envelope.data)
}

export const fetchDayEmailPreference = () =>
  call("/notifications/day-email", "GET")
export const updateDayEmailPreference = (frequency: 0 | 1 | 2) =>
  call("/notifications/day-email", "PUT", { frequency })
export const confirmDayEmail = (token: string) =>
  call("/notifications/day-email/confirm", "POST", { token })
