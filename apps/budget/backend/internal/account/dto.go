package account

import (
	"time"

	"budget_me/backend/internal/db"
)

// deletionResponse tells the client where the account stands. Both fields are
// null when no deletion is pending — the app shows nothing. Otherwise `deleteAt`
// is when cmd/worker will actually remove it, and the UI offers to cancel until
// then.
type deletionResponse struct {
	RequestedAt *string `json:"requestedAt"`
	DeleteAt    *string `json:"deleteAt"`
}

func toDeletionResponse(u db.User, grace time.Duration) deletionResponse {
	if !u.DeletionRequestedAt.Valid {
		return deletionResponse{}
	}

	requested := u.DeletionRequestedAt.Time.UTC().Format(time.RFC3339)
	deleteAt := u.DeletionRequestedAt.Time.Add(grace).UTC().Format(time.RFC3339)

	return deletionResponse{RequestedAt: &requested, DeleteAt: &deleteAt}
}
