import { z } from "zod"

import { isValidChoice, providerIds, type ProviderId } from "./catalog"

/** What the backend returns: the choice, and only a hint of the key. */
export const aiSettingsSchema = z.object({
  provider: z.enum(providerIds),
  model: z.string(),
  level: z.string(),
  keyHint: z.string().nullable(),
})
export type AiSettings = z.infer<typeof aiSettingsSchema>

/** An omitted `apiKey` keeps the stored one. */
export const aiSettingsInputSchema = z
  .object({
    provider: z.enum(providerIds),
    model: z.string().min(1),
    level: z.string().min(1),
    apiKey: z.string().trim().max(500).optional(),
  })
  .refine((input) => isValidChoice(input.provider, input.model, input.level))
export type AiSettingsInput = z.infer<typeof aiSettingsInputSchema>

/**
 * The key is only mandatory when none is stored for the chosen provider (or the
 * user is replacing it); the messages are keys of the `ai.errors` translations.
 */
export function makeAiFormSchema({
  storedProvider,
  replacing,
}: {
  storedProvider: ProviderId | null
  replacing: boolean
}) {
  return z
    .object({
      provider: z.enum(providerIds),
      model: z.string(),
      level: z.string(),
      apiKey: z.string().trim().max(500, "keyLong"),
    })
    .superRefine((values, ctx) => {
      if (!isValidChoice(values.provider, values.model, values.level)) {
        ctx.addIssue({ code: "custom", path: ["level"], message: "choice" })
      }

      const keyRequired = storedProvider !== values.provider || replacing

      if (keyRequired && values.apiKey === "") {
        ctx.addIssue({
          code: "custom",
          path: ["apiKey"],
          message: "keyRequired",
        })
      } else if (values.apiKey !== "" && values.apiKey.length < 8) {
        ctx.addIssue({ code: "custom", path: ["apiKey"], message: "keyShort" })
      }
    })
}
export type AiFormValues = z.infer<ReturnType<typeof makeAiFormSchema>>
