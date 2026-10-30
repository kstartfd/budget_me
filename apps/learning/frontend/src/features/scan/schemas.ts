import { z } from "zod"

/** A word or phrase the reader found in a photo, with its translation. */
export const foundSchema = z.object({
  kind: z.enum(["word", "phrase"]),
  text: z.string(),
  translation: z.string(),
})
export type Found = z.infer<typeof foundSchema>

export const scanSchema = z.object({ items: foundSchema.array() })
