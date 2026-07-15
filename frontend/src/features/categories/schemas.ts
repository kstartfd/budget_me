import { z } from "zod"

/**
 * Form shape for the add/edit category form (RHF + Zod). Shape validation only
 * — a non-empty name and a hex colour; the palette picker guarantees the hue,
 * and business rules (which side, preset locking) live in the screen/hooks.
 */
export const categoryFormSchema = z.object({
  name: z.string().trim().min(1),
  color: z.string().regex(/^#[0-9a-f]{6}$/i),
})
export type CategoryFormValues = z.infer<typeof categoryFormSchema>
