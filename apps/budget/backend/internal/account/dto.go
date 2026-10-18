package account

import (
	"time"

	ident "budget_me/account"
)

// deletionResponse tells the client where the account stands. Both fields are
// null when no deletion is pending — the app shows nothing. Otherwise `deleteAt`
// is when the account will actually be erased, and the UI offers to cancel until
// then.
type deletionResponse struct {
	RequestedAt *string `json:"requestedAt"`
	DeleteAt    *string `json:"deleteAt"`
}

func toDeletionResponse(acc ident.Account, grace time.Duration) deletionResponse {
	if acc.DeletionRequestedAt == nil {
		return deletionResponse{}
	}

	requested := acc.DeletionRequestedAt.UTC().Format(time.RFC3339)
	deleteAt := acc.DeletionRequestedAt.Add(grace).UTC().Format(time.RFC3339)

	return deletionResponse{RequestedAt: &requested, DeleteAt: &deleteAt}
}
