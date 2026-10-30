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
	// NoVision marks a model that cannot read a picture.
	NoVision bool
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
		{ID: "deepseek-v4-pro", Levels: deepseekLevels, NoVision: true},
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

// readLevel is the effort for reading a picture: finding the words in it needs
// no deliberation, and the learner is waiting.
var readLevel = map[Provider]string{Anthropic: "low", OpenAI: "low", DeepSeek: "off"}

// Reader is the model and effort that read a picture for a learner whose agent
// is `model`. A model that sees pictures reads them itself; one that does not is
// swapped for the first of the provider's models that does, which the same API
// key can use. False when the provider has none.
func Reader(provider Provider, model string) (readerModel, level string, ok bool) {
	level = readLevel[provider]

	for _, m := range catalog[provider] {
		if m.ID == model && !m.NoVision {
			return model, level, true
		}
	}

	for _, m := range catalog[provider] {
		if !m.NoVision {
			return m.ID, level, true
		}
	}

	return "", "", false
}
