import { readToken } from "@/features/auth"

import {
  createItemInputSchema,
  createListInputSchema,
  itemListSchema,
  itemSchema,
  updateItemInputSchema,
} from "./schemas"
import type {
  CreateItemInput,
  CreateListInput,
  Item,
  ItemList,
  UpdateItemInput,
} from "./schemas"

/**
 * Lists API — live against the Go backend (same-origin /api/v1). Own small
 * call()/ApiError, mirroring settings/tasks — each feature keeps its own
 * rather than sharing one, so nothing here risks an import cycle.
 */

class ApiError extends Error {
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
  const token = readToken()
  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })
  const body = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!body || body.code !== 0) {
    throw new ApiError(
      body?.message ?? res.statusText,
      body?.code ?? -1,
      res.status,
    )
  }
  return parse(body.data)
}

export async function fetchLists(): Promise<ItemList[]> {
  return call("/lists", { method: "GET" }, (d) => itemListSchema.array().parse(d))
}

export async function createList(input: CreateListInput): Promise<ItemList> {
  const parsed = createListInputSchema.parse(input)

  return call(
    "/lists",
    { method: "POST", body: JSON.stringify(parsed) },
    (d) => itemListSchema.parse(d),
  )
}

export async function deleteList(id: string): Promise<{ id: string }> {
  return call(`/lists/${id}`, { method: "DELETE" }, (d) => d as { id: string })
}

export async function fetchListItems(listId: string): Promise<Item[]> {
  return call(`/items?listId=${listId}`, { method: "GET" }, (d) =>
    itemSchema.array().parse(d),
  )
}

export async function createItem(input: CreateItemInput): Promise<Item> {
  const parsed = createItemInputSchema.parse(input)

  return call(
    "/items",
    { method: "POST", body: JSON.stringify(parsed) },
    (d) => itemSchema.parse(d),
  )
}

export async function updateItem(input: UpdateItemInput): Promise<Item> {
  const parsed = updateItemInputSchema.parse(input)

  return call(
    `/items/${parsed.id}`,
    {
      method: "PUT",
      body: JSON.stringify({
        title: parsed.title,
        purpose: parsed.purpose,
        verdict: parsed.verdict,
        rating: parsed.rating,
        note: parsed.note,
      }),
    },
    (d) => itemSchema.parse(d),
  )
}

export async function deleteItem(id: string): Promise<{ id: string }> {
  return call(`/items/${id}`, { method: "DELETE" }, (d) => d as { id: string })
}
