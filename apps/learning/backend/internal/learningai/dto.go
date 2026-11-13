package learningai

import (
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningai/commands"
)

type settingsRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Level    string `json:"level"`
	APIKey   string `json:"apiKey"`
}

type completeRequest struct {
	System string `json:"system"`
	Prompt string `json:"prompt"`
}

type completeResponse struct {
	Text string `json:"text"`
}

type settingsResponse struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Level    string `json:"level"`
	KeyHint  string `json:"keyHint"`
}

func settingsDTO(row db.LearningAiSetting) settingsResponse {
	return settingsResponse{Provider: string(row.Provider), Model: row.Model, Level: string(row.Level), KeyHint: row.KeyHint}
}

// scanResponse is what a picture held, for the learner to pick from. Known is
// how many of its texts the topic has already, Seen that the picture itself was
// read before.
type scanResponse struct {
	Items []commands.Found `json:"items"`
	Known int              `json:"known"`
	Seen  bool             `json:"seen"`
}
