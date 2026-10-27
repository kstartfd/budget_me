import {
  aiSettingsInputSchema,
  aiSettingsSchema,
  type AiSettings,
  type AiSettingsInput,
} from "./schemas"

const storageKey = "learning_app.ai.v1"

/**
 * Mock layer until the backend is wired: the choice lives in this browser.
 * The key itself is never kept, only its last four characters, which is all
 * the real endpoint will ever send back. Swap these two for GET/PUT
 * `/ai/settings` requests; the hooks stay as they are.
 */
export async function fetchAiSettings(): Promise<AiSettings | null> {
  return readStored()
}

export async function saveAiSettings(
  input: AiSettingsInput,
): Promise<AiSettings> {
  const body = aiSettingsInputSchema.parse(input)
  const previous = readStored()
  const key = body.apiKey?.trim()
  const keyHint = key
    ? key.slice(-4)
    : previous?.provider === body.provider
      ? previous.keyHint
      : null

  if (!keyHint) throw new Error("API key required")

  const saved: AiSettings = {
    provider: body.provider,
    model: body.model,
    level: body.level,
    keyHint,
  }
  writeStored(saved)

  return saved
}

function readStored(): AiSettings | null {
  try {
    const raw = localStorage.getItem(storageKey)
    const parsed = aiSettingsSchema.safeParse(raw ? JSON.parse(raw) : null)

    return parsed.success ? parsed.data : null
  } catch {
    return null
  }
}

function writeStored(settings: AiSettings) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(settings))
  } catch {
    // Storage can be blocked (private mode); the choice then lasts the session.
  }
}
