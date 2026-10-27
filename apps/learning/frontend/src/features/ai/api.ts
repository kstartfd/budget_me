import { call } from "@/features/learning/api"

import {
  aiSettingsInputSchema,
  aiSettingsSchema,
  type AiSettings,
  type AiSettingsInput,
} from "./schemas"

/** `null` until the user has chosen an agent. */
export const fetchAiSettings = (): Promise<AiSettings | null> =>
  call("/ai/settings", "GET", (d) => aiSettingsSchema.nullable().parse(d))

/** Omitting `apiKey` keeps the stored key (same provider only). */
export const saveAiSettings = (input: AiSettingsInput): Promise<AiSettings> =>
  call(
    "/ai/settings",
    "PUT",
    (d) => aiSettingsSchema.parse(d),
    aiSettingsInputSchema.parse(input),
  )

export const deleteAiSettings = (): Promise<void> =>
  call("/ai/settings", "DELETE", () => undefined)
