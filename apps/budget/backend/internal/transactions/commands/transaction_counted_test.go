package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/testdb"
	"budget_me/backend/internal/transactions/commands"
)

// Правило «из расчёта убирают только расход» проверяется ДО обращения к БД, поэтому
// хендлеры тут получают nil вместо запросов: дойди они до SQL — тест упал бы с
// паникой, а не с ошибкой.
func TestIncomeCannotBeLeftOutOfCalculation(t *testing.T) {
	at := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	create := commands.NewCreateHandler(nil, testdb.Clock{At: at})
	update := commands.NewUpdateHandler(nil)

	_, err := create.Handle(context.Background(), commands.CreateTransaction{
		UserID:      uuid.New(),
		CategoryID:  uuid.New(),
		AmountMinor: 100,
		Title:       "salary",
		SpentAt:     at,
		Direction:   db.TxDirectionIncome,
		Counted:     false,
	})
	if !errors.Is(err, commands.ErrIncomeAlwaysCounted) {
		t.Errorf("create: err = %v, want ErrIncomeAlwaysCounted", err)
	}

	_, err = update.Handle(context.Background(), commands.UpdateTransaction{
		UserID:      uuid.New(),
		ID:          uuid.New(),
		CategoryID:  uuid.New(),
		AmountMinor: 100,
		Title:       "salary",
		SpentAt:     at,
		Direction:   db.TxDirectionIncome,
		Counted:     false,
	})
	if !errors.Is(err, commands.ErrIncomeAlwaysCounted) {
		t.Errorf("update: err = %v, want ErrIncomeAlwaysCounted", err)
	}
}
