package settings

type settingsResponse struct {
	MonthStartDay int    `json:"monthStartDay"`
	Currency      string `json:"currency"`
	Today         string `json:"today"`
}

type updateSettingsRequest struct {
	MonthStartDay *int    `json:"monthStartDay"`
	Currency      *string `json:"currency"`
}
