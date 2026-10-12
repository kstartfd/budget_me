package auth

import (
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
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
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

func toUserResponse(u db.User) userResponse {
	res := userResponse{
		ID:           ids.FromPg(u.ID).String(),
		FirstName:    u.FirstName,
		Username:     db.TextPtr(u.Username),
		Email:        db.TextPtr(u.Email),
		TelegramID:   db.Int8Ptr(u.TelegramID),
		LanguageCode: db.TextPtr(u.LanguageCode),
	}

	return res
}
