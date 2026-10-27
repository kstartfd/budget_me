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
const gptLevels = ["none", "low", "medium", "high", "xhigh", "max"] as const

/**
 * The agents a user can pick. Model ids and the levels each accepts are the
 * provider's own (the backend sends them as-is), so a model that gains or
 * loses a level is a change here and nowhere else.
 */
export const providers: readonly ProviderSpec[] = [
  {
    id: "anthropic",
    label: "Claude",
    badge: "C",
    keyPlaceholder: "sk-ant-…",
    models: [
      {
        id: "claude-opus-5-5",
        label: "Opus 5.5",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "medium",
      },
      {
        id: "claude-sonnet-5-5",
        label: "Sonnet 5.5",
        kind: "effort",
        levels: claudeLevels,
        defaultLevel: "high",
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
        defaultLevel: "high",
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
        id: "gpt-5.6-terra",
        label: "GPT-5.6 Terra",
        kind: "effort",
        levels: gptLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-5.6-sol",
        label: "GPT-5.6 Sol",
        kind: "effort",
        levels: gptLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-5.6-luna",
        label: "GPT-5.6 Luna",
        kind: "effort",
        levels: gptLevels,
        defaultLevel: "medium",
      },
      {
        id: "gpt-5.5",
        label: "GPT-5.5",
        kind: "effort",
        levels: ["none", "low", "medium", "high", "xhigh"],
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
        id: "deepseek-v4-pro",
        label: "V4 Pro",
        kind: "thinking",
        levels: ["off", "high", "max"],
        defaultLevel: "high",
      },
      {
        id: "deepseek-v4-flash",
        label: "V4 Flash",
        kind: "thinking",
        levels: ["off", "high", "max"],
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
