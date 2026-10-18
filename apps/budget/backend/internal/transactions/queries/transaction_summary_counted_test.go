package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/testdb"
	txcmd "budget_me/backend/internal/transactions/commands"
	"budget_me/backend/internal/transactions/queries"
)

// addUncountedTx сеет расход с выключенной галочкой «в расчёте» — обязательный
// платёж, который остаётся в статистике, но не вычитается из денег.
func addUncountedTx(t *testing.T, q *db.Queries, userID, categoryID uuid.UUID, amountMinor int64, at time.Time) {
	t.Helper()

	h := txcmd.NewCreateHandler(q, testdb.Clock{At: at})
	if _, err := h.Handle(context.Background(), txcmd.CreateTransaction{
		UserID:      userID,
		CategoryID:  categoryID,
		AmountMinor: amountMinor,
		Title:       "seed",
		SpentAt:     at,
		Direction:   db.TxDirectionExpense,
		Counted:     false,
	}); err != nil {
		t.Fatalf("seed uncounted tx: %v", err)
	}
}

// TestSummarySplitsStatisticsFromCalculation — SpentMinor и разбивка по категориям
// видят ВСЕ траты (статистика), CountedSpentMinor — только те, что в расчёте
// («Свободно», полоска на дашборде).
func TestSummarySplitsStatisticsFromCalculation(t *testing.T) {
	pool, q := testdb.Pool(t)
	userID := testdb.NewUser(t, pool, q, jul(1), 1)

	addTx(t, q, userID, presetSalary, 500_000, db.TxDirectionIncome, jul(4))
	addTx(t, q, userID, presetGroceries, 10_000, db.TxDirectionExpense, jul(5))
	addUncountedTx(t, q, userID, presetGroceries, 30_000, jul(6))

	h := queries.NewSummaryHandler(q, testdb.Clock{At: jul(20)})

	got, err := h.Handle(context.Background(), queries.TransactionSummary{
		UserID: userID, From: rfc(jul(1)), To: rfc(jul(31)),
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if got.SpentMinor != 40_000 {
		t.Errorf("spent = %d, want 40000 (statistics see every expense)", got.SpentMinor)
	}

	if got.CountedSpentMinor != 10_000 {
		t.Errorf("counted spent = %d, want 10000 (the uncounted 30000 stays out)", got.CountedSpentMinor)
	}

	if got.IncomeMinor != 500_000 {
		t.Errorf("income = %d, want 500000", got.IncomeMinor)
	}

	if len(got.ByCategory) != 1 || got.ByCategory[0].SpentMinor != 40_000 {
		t.Errorf("by category = %+v, want one row of 40000", got.ByCategory)
	}
}

// TestCarriedInIgnoresUncountedExpenses — перенос в новый период считается от
// РАСЧЁТНОГО расхода: обязательный платёж с выключенной галочкой не съедает то, что
// переходит дальше.
func TestCarriedInIgnoresUncountedExpenses(t *testing.T) {
	pool, q := testdb.Pool(t)
	userID := testdb.NewUser(t, pool, q, jul(1), 1)

	jun := func(day int) time.Time { return time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC) }

	addTx(t, q, userID, presetSalary, 200_000, db.TxDirectionIncome, jun(5))
	addTx(t, q, userID, presetGroceries, 50_000, db.TxDirectionExpense, jun(10))
	addUncountedTx(t, q, userID, presetGroceries, 80_000, jun(12))

	h := queries.NewSummaryHandler(q, testdb.Clock{At: jul(20)})

	got, err := h.Handle(context.Background(), queries.TransactionSummary{
		UserID:   userID,
		From:     rfc(jul(1)),
		To:       rfc(jul(31)),
		PrevFrom: rfc(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
		PrevTo:   rfc(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if got.CarriedInMinor != 150_000 {
		t.Errorf("carried in = %d, want 150000 (200000 income − 50000 counted)", got.CarriedInMinor)
	}
}
