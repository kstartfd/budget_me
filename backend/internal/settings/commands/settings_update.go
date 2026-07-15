package commands

import (
	"context"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type UpdateSettings struct {
	UserID        uuid.UUID
	MonthStartDay *int
	Currency      *string
}

type Settings struct {
	MonthStartDay int
	Currency      string
	Today         string
}

type UpdateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewUpdateHandler(q *db.Queries, clk clock.Clock) UpdateHandler {
	return UpdateHandler{q: q, clock: clk}
}

func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateSettings) (Settings, error) {
	u, err := h.q.GetUserByID(ctx, ids.ToPg(cmd.UserID))
	if err != nil {
		return Settings{}, err
	}

	day := int(u.MonthStartDay)
	if cmd.MonthStartDay != nil {
		day = *cmd.MonthStartDay
	}

	if day < 1 || day > 28 {
		return Settings{}, apperr.NewInvalid("month start day must be between 1 and 28")
	}

	currency := u.Currency

	if cmd.Currency != nil {
		if !validCurrency(*cmd.Currency) {
			return Settings{}, apperr.NewInvalid("unsupported currency")
		}

		currency = db.CurrencyCode(*cmd.Currency)
	}

	updated, err := h.q.UpdateUserSettings(ctx, db.UpdateUserSettingsParams{
		ID:            ids.ToPg(cmd.UserID),
		MonthStartDay: int16(day),
		Currency:      currency,
		Now:           db.Time(h.clock.Now()),
	})
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		MonthStartDay: int(updated.MonthStartDay),
		Currency:      string(updated.Currency),
		Today:         h.clock.Now().UTC().Format("2006-01-02"),
	}, nil
}

func validCurrency(c string) bool {
	switch c {
	case "BYN", "USD", "EUR", "RUB":
		return true
	default:
		return false
	}
}
