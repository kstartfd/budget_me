package learningai

import "learning_app/backend/internal/db"

type settingsRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Level    string `json:"level"`
	APIKey   string `json:"apiKey"`
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
