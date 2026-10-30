package learningai

import (
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningai/commands"
)

type settingsRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Level    string `json:"level"`
	Teacher  bool   `json:"teacher"`
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
	Teacher  bool   `json:"teacher"`
	KeyHint  string `json:"keyHint"`
}

func settingsDTO(row db.LearningAiSetting) settingsResponse {
	return settingsResponse{Provider: string(row.Provider), Model: row.Model, Level: string(row.Level), Teacher: row.Teacher, KeyHint: row.KeyHint}
}

// scanResponse is what a picture held, for the learner to pick from.
type scanResponse struct {
	Items []commands.Found `json:"items"`
}
