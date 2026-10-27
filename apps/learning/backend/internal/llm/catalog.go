package llm

import "slices"

type Provider string

const (
	Anthropic Provider = "anthropic"
	OpenAI    Provider = "openai"
	DeepSeek  Provider = "deepseek"
)

type Model struct {
	ID     string
	Levels []string
}

var (
	claudeLevels   = []string{"low", "medium", "high", "xhigh", "max"}
	gptLevels      = []string{"low", "medium", "high", "xhigh", "max"}
	gptLightLevels = []string{"none", "low", "medium", "high", "xhigh", "max"}
	deepseekLevels = []string{"off", "low", "high", "max"}
)

var catalog = map[Provider][]Model{
	Anthropic: {
		{ID: "claude-opus-5-5", Levels: claudeLevels},
		{ID: "claude-sonnet-5-5", Levels: claudeLevels},
		{ID: "claude-haiku-5-5", Levels: claudeLevels},
		{ID: "claude-fable-5-1", Levels: claudeLevels},
	},
	OpenAI: {
		{ID: "gpt-6-astra", Levels: gptLevels},
		{ID: "gpt-6.1-sol", Levels: gptLevels},
		{ID: "gpt-6.1-terra", Levels: gptLightLevels},
		{ID: "gpt-6-luna", Levels: gptLightLevels},
	},
	DeepSeek: {
		{ID: "deepseek-v4-pro", Levels: deepseekLevels},
		{ID: "deepseek-flash", Levels: deepseekLevels},
	},
}

func Valid(provider Provider, model, level string) bool {
	for _, m := range catalog[provider] {
		if m.ID == model {
			return slices.Contains(m.Levels, level)
		}
	}

	return false
}
