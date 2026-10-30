import { readToken } from "@/features/auth"
import { call } from "@/features/learning/api"

import { scanSchema, type Found, type Scanned } from "./schemas"

/** Business codes of the backend that reading a photo tells apart. */
export const scanErrorCode = {
  /** Wrong input (a file that is no picture, a language that cannot be read): the message says which, in the user's language. */
  invalid: 10001,
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

/** Has the backend read the words and phrases written in a photo. */
export async function scanPicture(
  topicId: string,
  image: Blob,
): Promise<Scanned> {
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
    throw new ScanError("network")
  })
  const body = await res.json().catch(() => null)

  if (!res.ok || body?.code !== 0)
    throw new ScanError(body?.message ?? res.statusText, body?.code)

  return scanSchema.parse(body.data)
}

/**
 * Saves a found word or phrase the way the browser extension does: as a
 * capture, which the backend turns into an entry and, with an agent, fills in.
 */
export const enqueueFound = (topicId: string, found: Found, source: string) =>
  call("/captures", "POST", () => undefined, {
    topicId,
    kind: found.kind,
    text: found.text,
    source,
    status: "inbox",
  })
