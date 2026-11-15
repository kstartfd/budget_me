import { readToken } from "@/features/auth"
import {
  entryInputSchema,
  entrySchema,
  issuedTokenSchema,
  planSchema,
  reminderSchema,
  reviewSchema,
  setsSchema,
  tokenInputSchema,
  tokenSchema,
  tokenSecretSchema,
  topicInputSchema,
  topicSchema,
  type EntryInput,
  type Grade,
  type Plan,
  type TokenInput,
  type TopicInput,
} from "./schemas"

export async function call<T>(
  path: string,
  method: string,
  parse: (data: unknown) => T,
  data?: unknown,
): Promise<T> {
  const token = readToken()
  const res = await fetch(`/api/v1/learning${path}`, {
    method,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(data !== undefined ? { body: JSON.stringify(data) } : {}),
  })
  const body = await res.json().catch(() => null)
  if (!res.ok || body?.code !== 0)
    throw new Error(body?.message ?? res.statusText)
  return parse(body.data)
}

export const fetchTopics = () =>
  call("/topics", "GET", (d) => topicSchema.array().parse(d))
export const saveTopic = (input: TopicInput & { id?: string }) =>
  call(
    input.id ? `/topics/${input.id}` : "/topics",
    input.id ? "PUT" : "POST",
    (d) => topicSchema.parse(d),
    topicInputSchema.parse(input),
  )
export const deleteTopic = (id: string) =>
  call(`/topics/${id}`, "DELETE", () => undefined)
export const PAGE_SIZE = 50

export const fetchEntries = (
  topicId: string,
  offset: number,
  filters: { kind: string; status: string; search: string },
) =>
  call(
    `/entries?${new URLSearchParams({ topicId, limit: String(PAGE_SIZE), offset: String(offset), kind: filters.kind === "all" ? "" : filters.kind, status: filters.status === "all" ? "" : filters.status, search: filters.search })}`,
    "GET",
    (d) => entrySchema.array().parse(d),
  )
export const saveEntry = (input: EntryInput & { id?: string }) =>
  call(
    input.id ? `/entries/${input.id}` : "/entries",
    input.id ? "PUT" : "POST",
    (d) => entrySchema.parse(d),
    entryInputSchema.parse(input),
  )
/** Has the learner's AI agent write the fields the entry still lacks. */
export const fillEntry = (id: string) =>
  call(`/entries/${id}/fill`, "POST", (d) => entrySchema.parse(d))
/** Has the learner's AI agent write a pack of entries into a topic. */
export const generatePack = (topicId: string, kind: "phrases" | "words") =>
  call(`/topics/${topicId}/pack`, "POST", (d) => entrySchema.array().parse(d), {
    kind,
  })
/** The cards due for review in a topic. */
export const fetchReview = (topicId: string) =>
  call(`/topics/${topicId}/review`, "GET", (d) => reviewSchema.parse(d))
/** Records how a review of one entry went; the server moves it along its schedule. */
export const reviewEntry = (id: string, grade: Grade) =>
  call(`/entries/${id}/review`, "POST", (d) => entrySchema.parse(d), { grade })
/**
 * Saves a typed word or phrase the way the browser extension and the photo scan
 * do: as a capture, which the backend turns into an entry and, with an agent,
 * fills in the background.
 */
export const enqueueCapture = (input: {
  topicId: string
  kind: "word" | "phrase"
  text: string
}) =>
  call("/captures", "POST", () => undefined, {
    ...input,
    source: "",
    status: "inbox",
  })
const zone = () => Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"

/** Opens the topic's daily sets for the learner's today. */
export const fetchSets = (topicId: string) =>
  call(`/topics/${topicId}/sets?tz=${encodeURIComponent(zone())}`, "GET", (d) =>
    setsSchema.parse(d),
  )
/** The next batch of a weekly or monthly review. */
export const fetchSetPeriod = (topicId: string, period: "week" | "month") =>
  call(
    `/topics/${topicId}/sets/${period}?tz=${encodeURIComponent(zone())}`,
    "GET",
    (d) => entrySchema.array().parse(d),
  )
export const savePlan = (topicId: string, plan: Plan) =>
  call(`/topics/${topicId}/plan`, "PUT", (d) => planSchema.parse(d), plan)
/** How a card went in a set: known or not. */
export const gradeSetEntry = (
  id: string,
  input: { period: "day" | "week" | "month"; known: boolean },
) =>
  call(`/entries/${id}/set-grade`, "POST", (d) => entrySchema.parse(d), {
    ...input,
    tz: zone(),
  })
export const fetchReminder = () =>
  call("/reminder", "GET", (d) => reminderSchema.parse(d))
export const saveReminder = (input: { enabled: boolean; timezone: string }) =>
  call("/reminder", "PUT", (d) => reminderSchema.parse(d), input)
export const fetchEntry = (id: string) =>
  call(`/entries/${id}`, "GET", (d) => entrySchema.parse(d))
export async function fetchEntryAudio(id: string): Promise<Blob> {
  const token = readToken()
  const res = await fetch(`/api/v1/learning/entries/${id}/audio`, {
    cache: "no-store",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.message ?? res.statusText)
  }
  return res.blob()
}
export const deleteEntry = (id: string) =>
  call(`/entries/${id}`, "DELETE", () => undefined)
export const fetchTokens = () =>
  call("/tokens", "GET", (d) => tokenSchema.array().parse(d))
export const createToken = (input: TokenInput) =>
  call(
    "/tokens",
    "POST",
    (d) => issuedTokenSchema.parse(d),
    tokenInputSchema.parse(input),
  )
export const fetchTokenSecret = (id: string) =>
  call(`/tokens/${id}/secret`, "GET", (d) => tokenSecretSchema.parse(d).token)
export const revokeToken = (id: string) =>
  call(`/tokens/${id}`, "DELETE", () => undefined)
