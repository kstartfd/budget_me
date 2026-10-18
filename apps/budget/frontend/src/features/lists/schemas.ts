import { z } from "zod"

/**
 * A list (Аптечка, Косметика, or a personal one) is a dictionary — mirrors
 * categories exactly: shared presets carry a `slug` (display name resolved by
 * i18n) and no `name`; personal lists carry a free `name` and no `slug`.
 *
 * An item belongs to exactly one list. `rating` is 1..10 (rendered as 5 stars
 * with halves — see components/star-rating.tsx); null means not rated yet,
 * which is different from `verdict: "unknown"` (rated 0 stars is not a thing
 * here — you either haven't tried it, or you have an opinion).
 */

export const verdictSchema = z.enum(["works", "meh", "no", "unknown"])
export type Verdict = z.infer<typeof verdictSchema>

export const itemListSchema = z.object({
  id: z.string(),
  slug: z.string().nullable(),
  name: z.string().nullable(),
  color: z.string(),
  /** How many items the list holds — shown on its card once GET /lists sends it. */
  itemCount: z.number().int().nonnegative().optional(),
})
export type ItemList = z.infer<typeof itemListSchema>

export const itemSchema = z.object({
  id: z.string(),
  listId: z.string(),
  title: z.string(),
  purpose: z.string().nullable(),
  verdict: verdictSchema,
  rating: z.number().min(1).max(10).nullable(),
  note: z.string().nullable(),
  photoUrl: z.string().nullable(),
  createdAt: z.string(),
})
export type Item = z.infer<typeof itemSchema>

export const createListInputSchema = z.object({
  name: z.string().trim().min(1).max(60),
  color: z.string().regex(/^#[0-9a-f]{6}$/i),
})
export type CreateListInput = z.infer<typeof createListInputSchema>

/** `PUT /lists/:id` — personal lists only; a preset 404s (see the backend). */
export const updateListInputSchema = z.object({
  id: z.string(),
  name: z.string().trim().min(1).max(60),
  color: z.string().regex(/^#[0-9a-f]{6}$/i),
})
export type UpdateListInput = z.infer<typeof updateListInputSchema>

export const createItemInputSchema = z.object({
  listId: z.string(),
  title: z.string().trim().min(1).max(200),
  purpose: z.string().trim().max(200).nullable(),
  verdict: verdictSchema,
  rating: z.number().min(1).max(10).nullable(),
  note: z.string().trim().max(500).nullable(),
})
export type CreateItemInput = z.infer<typeof createItemInputSchema>

/** `PUT /items/:id` replaces every field wholesale, like tasks — no PATCH. */
export const updateItemInputSchema = z.object({
  id: z.string(),
  title: z.string().trim().min(1).max(200),
  purpose: z.string().trim().max(200).nullable(),
  verdict: verdictSchema,
  rating: z.number().min(1).max(10).nullable(),
  note: z.string().trim().max(500).nullable(),
})
export type UpdateItemInput = z.infer<typeof updateItemInputSchema>
