export const providerIds = ["anthropic", "openai", "deepseek"] as const
export type ProviderId = (typeof providerIds)[number]

/**
 * What the third step of the picker is for a model: how hard it reasons
 * ("effort") or whether it reasons at all and how far ("thinking").
 */
export type LevelKind = "effort" | "thinking"

export interface ModelSpec {
  id: string
  label: string
  kind: LevelKind
  levels: readonly string[]
  defaultLevel: string
}

export interface ProviderSpec {
  id: ProviderId
  label: string
  badge: string
  keyPlaceholder: string
  models: readonly ModelSpec[]
}

const claudeLevels = ["low", "medium", "high", "xhigh", "max"] as const
const gptLevels = ["low", "medium", "high", "xhigh", "max"] as const
const gptLightLevels = ["none", ...gptLevels] as const
const deepseekLevels = ["off", "low", "high", "max"] as const

/**
 * The agents a user can pick; the backend keeps the same list and rejects
 * anything else (`internal/llm/catalog.go`). The first model of each provider
 * is what a fresh pick lands on, so it is the mid-tier one: a card for one
 * saved word does not need the flagship.
 */
export const providers: readonly ProviderSpec[] = [
  {
    id: "anthropic",
    label: "Claude",
    badge: "C",
    keyPlaceholder: "sk-ant-…",
    models: [
      {
        id: "claude-sonnet-5-5",
        label: "Sonnet 5.5",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "medium",
      },
      {
        id: "claude-opus-5-5",
        label: "Opus 5.5",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "medium",
      },
      {
        id: "claude-haiku-5-5",
        label: "Haiku 5.5",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "medium",
      },
      {
        id: "claude-fable-5-1",
        label: "Fable 5.1",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "medium",
      },
    ],
  },
  {
    id: "openai",
    label: "ChatGPT",
    badge: "G",
    keyPlaceholder: "sk-…",
    models: [
      {
        id: "gpt-6.1-terra",
        label: "GPT-6.1 Terra",
        kind: "effort",
        levels: gptLightLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-6.1-sol",
        label: "GPT-6.1 Sol",
        kind: "effort",
        levels: gptLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-6-astra",
        label: "GPT-6 Astra",
        kind: "effort",
        levels: gptLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-6-luna",
        label: "GPT-6 Luna",
        kind: "effort",
        levels: gptLightLevels,
        defaultLevel: "medium",
      },
    ],
  },
  {
    id: "deepseek",
    label: "DeepSeek",
    badge: "D",
    keyPlaceholder: "sk-…",
    models: [
      {
        id: "deepseek-flash",
        label: "V4.1 Flash",
        kind: "thinking",
        levels: deepseekLevels,
        defaultLevel: "high",
      },
      {
        id: "deepseek-v4-pro",
        label: "V4 Pro",
        kind: "thinking",
        levels: deepseekLevels,
        defaultLevel: "high",
      },
    ],
  },
]

export function findProvider(id: string): ProviderSpec | undefined {
  return providers.find((provider) => provider.id === id)
}

export function findModel(
  providerId: string,
  modelId: string,
): ModelSpec | undefined {
  return findProvider(providerId)?.models.find((model) => model.id === modelId)
}

export function isValidChoice(
  providerId: string,
  modelId: string,
  level: string,
): boolean {
  return findModel(providerId, modelId)?.levels.includes(level) ?? false
}
