package auth

import (
	"time"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
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
	// DeleteAt is set only while a deletion is pending: the moment cmd/worker
	// will remove this account. The app reads it on every load to show the
	// "your account is scheduled for deletion — cancel?" banner.
	DeleteAt *string `json:"delete_at,omitempty"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

func toUserResponse(u db.User, deletionGrace time.Duration) userResponse {
	res := userResponse{
		ID:           ids.FromPg(u.ID).String(),
		FirstName:    u.FirstName,
		Username:     db.TextPtr(u.Username),
		Email:        db.TextPtr(u.Email),
		TelegramID:   db.Int8Ptr(u.TelegramID),
		LanguageCode: db.TextPtr(u.LanguageCode),
	}

	if u.DeletionRequestedAt.Valid {
		deleteAt := u.DeletionRequestedAt.Time.Add(deletionGrace).UTC().Format(time.RFC3339)
		res.DeleteAt = &deleteAt
	}

	return res
}
