import { readToken } from "@/features/auth"
import { call } from "@/features/learning/api"

import { scanSchema, type Found } from "./schemas"

/** Business codes of the backend that reading a photo tells apart. */
export const scanErrorCode = {
  /** Wrong input or no agent set up: the message says which, in the user's language. */
  invalid: 10001,
  rejected: 30001,
  unavailable: 30002,
  agentFailed: 30003,
  tooMany: 42901,
} as const

export class ScanError extends Error {
  constructor(
    message: string,
    readonly code?: number,
  ) {
    super(message)
    this.name = "ScanError"
  }
}

/** Asks the user's agent for the words and phrases written in a photo. */
export async function scanPicture(
  topicId: string,
  image: Blob,
): Promise<Found[]> {
  const token = readToken()
  const form = new FormData()
  form.set("topicId", topicId)
  form.set("image", image, "photo.jpg")

  // No Content-Type: the browser adds the multipart one with its boundary.
  const res = await fetch("/api/v1/learning/ai/scan", {
    method: "POST",
    cache: "no-store",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  }).catch(() => {
    throw new ScanError("network", scanErrorCode.unavailable)
  })
  const body = await res.json().catch(() => null)

  if (!res.ok || body?.code !== 0)
    throw new ScanError(body?.message ?? res.statusText, body?.code)

  return scanSchema.parse(body.data).items
}

/**
 * Saves a found word or phrase the way the browser extension does: as a
 * capture, which the backend turns into an entry and lets the agent fill in.
 */
export const enqueueFound = (topicId: string, found: Found, source: string) =>
  call("/captures", "POST", () => undefined, {
    topicId,
    kind: found.kind,
    text: found.text,
    translation: found.translation,
    source,
    status: "inbox",
  })
