package commands

import (
	"context"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// ErrHistoryResetRequired guards the one setting that cannot change quietly.
// Moving the day a billing period opens re-cuts every period the app has shown,
// so the client has to come back with ResetHistory set once the user has agreed
// to it. Conflict (409/40001) rather than Invalid: the request is well-formed,
// it is the current state that refuses it — and on the settings endpoint that is
// the only 409, which is how the client tells this apart from a bad field.
var ErrHistoryResetRequired = apperr.NewConflict("changing the month start day resets the running totals; confirm to continue")

type UpdateSettings struct {
	UserID        uuid.UUID
	MonthStartDay *int
	Currency      *string
	// ResetHistory is the user's answer to the confirmation, not a preference:
	// it only matters when MonthStartDay actually moves, and is ignored otherwise
	// so a stale client cannot wipe the running totals on a currency change.
	ResetHistory bool
}

type Settings struct {
	MonthStartDay  int
	Currency       string
	Today          string
	HistoryResetAt *time.Time
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

	// Re-sending the SAME day is not a change and must not cost the user their
	// running totals — only an actual move asks for, and acts on, confirmation.
	if day == int(u.MonthStartDay) {
		return h.save(ctx, cmd.UserID, day, currency, db.TimePtr(u.HistoryResetAt))
	}

	if !cmd.ResetHistory {
		return Settings{}, ErrHistoryResetRequired
	}

	return h.saveWithReset(ctx, cmd.UserID, day, currency)
}

// save is the ordinary path: this update leaves the cutoff alone, so an earlier
// reset stays in force and is echoed back as it was read at the top of Handle.
func (h UpdateHandler) save(
	ctx context.Context,
	userID uuid.UUID,
	day int,
	currency db.CurrencyCode,
	resetAt *time.Time,
) (Settings, error) {
	updated, err := h.q.UpdateUserSettings(ctx, db.UpdateUserSettingsParams{
		ID:            ids.ToPg(userID),
		MonthStartDay: int16(day),
		Currency:      currency,
		Now:           db.Time(h.clock.Now()),
	})
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		MonthStartDay:  int(updated.MonthStartDay),
		Currency:       string(updated.Currency),
		Today:          h.clock.Now().UTC().Format("2006-01-02"),
		HistoryResetAt: resetAt,
	}, nil
}

func (h UpdateHandler) saveWithReset(ctx context.Context, userID uuid.UUID, day int, currency db.CurrencyCode) (Settings, error) {
	now := h.clock.Now()

	updated, err := h.q.UpdateUserSettingsAndResetHistory(ctx, db.UpdateUserSettingsAndResetHistoryParams{
		ID:            ids.ToPg(userID),
		MonthStartDay: int16(day),
		Currency:      currency,
		Now:           db.Time(now),
	})
	if err != nil {
		return Settings{}, err
	}

	at := now.UTC()

	return Settings{
		MonthStartDay:  int(updated.MonthStartDay),
		Currency:       string(updated.Currency),
		Today:          now.UTC().Format("2006-01-02"),
		HistoryResetAt: &at,
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
