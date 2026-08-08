package settings

import "time"

// HistoryResetAt is null until the user has moved the month start day and
// accepted the reset; from then on it is the instant the running totals restart
// from (RFC3339 UTC).
type settingsResponse struct {
	MonthStartDay  int     `json:"monthStartDay"`
	Currency       string  `json:"currency"`
	Today          string  `json:"today"`
	HistoryResetAt *string `json:"historyResetAt"`
}

// ResetHistory carries the user's confirmation. Without it a changed
// monthStartDay is refused with 409 rather than applied.
type updateSettingsRequest struct {
	MonthStartDay *int    `json:"monthStartDay"`
	Currency      *string `json:"currency"`
	ResetHistory  bool    `json:"resetHistory"`
}

func resetAtString(at *time.Time) *string {
	if at == nil {
		return nil
	}

	s := at.UTC().Format(time.RFC3339)

	return &s
}
