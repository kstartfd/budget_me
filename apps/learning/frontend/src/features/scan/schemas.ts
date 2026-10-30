import { z } from "zod"

/** A word or phrase the backend read off a photo. */
export const foundSchema = z.object({
  kind: z.enum(["word", "phrase"]),
  text: z.string(),
})
export type Found = z.infer<typeof foundSchema>

/**
 * What a photo held. `known` counts the texts the topic already has (they are
 * not in `items`), `seen` says this very photo was sent for the topic before.
 */
export const scanSchema = z.object({
  items: foundSchema.array(),
  known: z.number(),
  seen: z.boolean(),
})
export type Scanned = z.infer<typeof scanSchema>
