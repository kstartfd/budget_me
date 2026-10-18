package auth

import (
	"time"

	"github.com/google/uuid"

	ident "budget_me/account"
)

type telegramRequest struct {
	InitData string `json:"init_data"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type linkEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID           string  `json:"id"`
	FirstName    string  `json:"first_name"`
	Username     *string `json:"username,omitempty"`
	Email        *string `json:"email,omitempty"`
	TelegramID   *int64  `json:"telegram_id,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	// DeleteAt is set only while a deletion is pending: when the account will be
	// erased. The app reads it on every load to show the cancel banner.
	DeleteAt *string `json:"delete_at,omitempty"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

// toUserResponse describes an account to its owner. The id is the app's own
// user id (the one the session token carries), not the account id.
func toUserResponse(userID uuid.UUID, acc ident.Account, grace time.Duration) userResponse {
	res := userResponse{
		ID:           userID.String(),
		FirstName:    acc.FirstName,
		Username:     acc.Username,
		Email:        acc.Email,
		TelegramID:   acc.TelegramID,
		LanguageCode: acc.LanguageCode,
	}

	if at := acc.DeleteAt(grace); at != nil {
		deleteAt := at.UTC().Format(time.RFC3339)
		res.DeleteAt = &deleteAt
	}

	return res
}
