package balance_test

import (
	// Стандартная библиотека.
	"context"
	"testing"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	balancecommands "budget_me/backend/internal/balance/commands"
	balancequeries "budget_me/backend/internal/balance/queries"
	"budget_me/backend/internal/db"
	txcommands "budget_me/backend/internal/transactions/commands"
)

// groceriesCategoryID — общая expense-категория из 0004_seed_categories.sql.
const groceriesCategoryID = "00000000-0000-4000-8000-000000000002"

func logExpense(t *testing.T, q *db.Queries, userID uuid.UUID, at time.Time, amountMinor int64, counted bool) {
	t.Helper()

	h := txcommands.NewCreateHandler(q, fakeClock{now: at})

	_, err := h.Handle(context.Background(), txcommands.CreateTransaction{
		UserID:      userID,
		CategoryID:  uuid.MustParse(groceriesCategoryID),
		AmountMinor: amountMinor,
		Title:       "expense",
		SpentAt:     at,
		Direction:   db.TxDirectionExpense,
		Counted:     counted,
	})
	if err != nil {
		t.Fatalf("log expense: %v", err)
	}
}

// TestBalanceIgnoresUncountedExpenses — обязательный платёж с выключенной галочкой
// «в расчёте» не вычитается из Остатка; обычная трата вычитается как раньше.
func TestBalanceIgnoresUncountedExpenses(t *testing.T) {
	pool, q := newTestPool(t)

	june := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	july := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	userID := newTestUser(t, pool, q, june)

	setHandler := balancecommands.NewSetHandler(q, fakeClock{now: june})
	if _, err := setHandler.Handle(context.Background(), balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 1_000_000,
	}); err != nil {
		t.Fatalf("set anchor: %v", err)
	}

	logExpense(t, q, userID, july, 100_000, true)
	logExpense(t, q, userID, july, 400_000, false)

	got, err := balancequeries.NewGetHandler(q).Handle(context.Background(), balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if want := int64(1_000_000 - 100_000); got.BalanceMinor != want {
		t.Errorf("BalanceMinor = %d, want %d — the uncounted 400000 must not be deducted", got.BalanceMinor, want)
	}
}
