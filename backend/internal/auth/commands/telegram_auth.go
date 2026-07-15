package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type AuthTelegram struct {
	InitData string
}

type AuthTelegramHandler struct {
	q     *db.Queries
	clock clock.Clock
	bot   string
}

func NewAuthTelegramHandler(q *db.Queries, clk clock.Clock, botToken string) AuthTelegramHandler {
	return AuthTelegramHandler{q: q, clock: clk, bot: botToken}
}

// defaultTimezone — пока клиент не сообщил свою (GET /settings?tz=). Нужна только
// фоновым задачам: запросы всегда несут зону с собой.
const defaultTimezone = "Europe/Minsk"

func (h AuthTelegramHandler) Handle(ctx context.Context, cmd AuthTelegram) (db.User, error) {
	tgUser, err := VerifyInitData(cmd.InitData, h.bot, h.clock.Now())
	if err != nil {
		return db.User{}, apperr.NewUnauthorized(err.Error())
	}

	name := strings.TrimSpace(tgUser.FirstName + " " + tgUser.LastName)
	existing, err := h.q.GetUserByTelegramID(ctx, db.Int8(&tgUser.ID))

	switch {
	case err == nil:
		return h.q.UpdateUserTelegramProfile(ctx, db.UpdateUserTelegramProfileParams{
			ID:           existing.ID,
			FirstName:    name,
			Username:     db.Text(optional(tgUser.Username)),
			LanguageCode: db.Text(optional(tgUser.LanguageCode)),
			UpdatedAt:    db.Time(h.clock.Now()),
		})
	case errors.Is(err, pgx.ErrNoRows):
		now := h.clock.Now()

		return h.q.InsertUser(ctx, db.InsertUserParams{
			ID:            ids.ToPg(ids.New()),
			TelegramID:    db.Int8(&tgUser.ID),
			FirstName:     name,
			Username:      db.Text(optional(tgUser.Username)),
			LanguageCode:  db.Text(optional(tgUser.LanguageCode)),
			MonthStartDay: 1,
			Currency:      db.CurrencyCode("BYN"),
			// Стартовая зона; клиент пришлёт свою при первом GET /settings.
			Timezone:  defaultTimezone,
			CreatedAt: db.Time(now),
			UpdatedAt: db.Time(now),
		})
	default:
		return db.User{}, err
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
