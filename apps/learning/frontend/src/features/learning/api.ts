import { readToken } from "@/features/auth"
import {
  entryInputSchema,
  entrySchema,
  issuedTokenSchema,
  tokenInputSchema,
  tokenSchema,
  topicInputSchema,
  topicSchema,
  type EntryInput,
  type TokenInput,
  type TopicInput,
} from "./schemas"

async function call<T>(
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
export const fetchEntries = (
  topicId: string,
  offset: number,
  filters: { kind: string; status: string; search: string },
) =>
  call(
    `/entries?${new URLSearchParams({ topicId, limit: "100", offset: String(offset), kind: filters.kind === "all" ? "" : filters.kind, status: filters.status === "all" ? "" : filters.status, search: filters.search })}`,
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
export const fetchEntry = (id: string) =>
  call(`/entries/${id}`, "GET", (d) => entrySchema.parse(d))
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
export const revokeToken = (id: string) =>
  call(`/tokens/${id}`, "DELETE", () => undefined)
