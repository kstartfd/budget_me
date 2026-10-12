package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	catcmd "budget_me/backend/internal/categories/commands"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/testdb"
	txcmd "budget_me/backend/internal/transactions/commands"
	"budget_me/backend/internal/transactions/queries"
)

// Шапка экрана «Транзакции» считает сумму под фильтром в SQL (COALESCE + FILTER
// по direction + необязательный category_id через sqlc.narg). Проверить это
// честно можно только против настоящей Postgres — sqlc IS the repository.

var (
	presetGroceries = uuid.MustParse(testdb.PresetExpenseCategory)
	presetSalary    = uuid.MustParse(testdb.PresetIncomeCategory)
)

func jul(day int) time.Time { return time.Date(2026, 7, day, 12, 0, 0, 0, time.UTC) }

func rfc(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// addTx сеет одну трату настоящей командой создания (она же сверит направление
// траты с направлением категории).
func addTx(t *testing.T, q *db.Queries, userID, categoryID uuid.UUID, amountMinor int64, dir db.TxDirection, at time.Time) {
	t.Helper()

	h := txcmd.NewCreateHandler(q, testdb.Clock{At: at})
	if _, err := h.Handle(context.Background(), txcmd.CreateTransaction{
		UserID:      userID,
		CategoryID:  categoryID,
		AmountMinor: amountMinor,
		Title:       "seed",
		SpentAt:     at,
		Direction:   dir,
	}); err != nil {
		t.Fatalf("seed tx: %v", err)
	}
}

// TestTotalsSplitSpendAndIncome — расход и приход в шапке считаются РАЗДЕЛЬНО по
// direction: приход не течёт в расход и наоборот.
func TestTotalsSplitSpendAndIncome(t *testing.T) {
	pool, q := testdb.Pool(t)
	userID := testdb.NewUser(t, pool, q, jul(1), 1)

	addTx(t, q, userID, presetGroceries, 10_000, db.TxDirectionExpense, jul(3))
	addTx(t, q, userID, presetGroceries, 25_000, db.TxDirectionExpense, jul(5))
	addTx(t, q, userID, presetSalary, 500_000, db.TxDirectionIncome, jul(4))

	h := queries.NewTotalsHandler(q, testdb.Clock{At: jul(20)})

	got, err := h.Handle(context.Background(), queries.TransactionTotals{
		UserID: userID,
		From:   rfc(jul(1)),
		To:     rfc(jul(31)),
	})
	if err != nil {
		t.Fatalf("totals: %v", err)
	}

	if got.SpentMinor != 35_000 {
		t.Errorf("spent = %d, want 35000", got.SpentMinor)
	}

	if got.IncomeMinor != 500_000 {
		t.Errorf("income = %d, want 500000", got.IncomeMinor)
	}
}

// TestTotalsFilterByCategory — с Category в сумму попадает только эта категория;
// с uuid.Nil (по всем) — все. Ровно то, ради чего заводили эндпоинт: цифра под
// ссылкой «из категории» и под фильтром категории.
func TestTotalsFilterByCategory(t *testing.T) {
	pool, q := testdb.Pool(t)
	userID := testdb.NewUser(t, pool, q, jul(1), 1)

	mk := catcmd.NewCreateHandler(q, testdb.Clock{At: jul(1)})

	cat, err := mk.Handle(context.Background(), catcmd.CreateCategory{
		UserID:    userID,
		Name:      "Transport",
		Color:     "#7c86ff",
		Direction: db.TxDirectionExpense,
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	transportID := ids.FromPg(cat.ID)

	addTx(t, q, userID, presetGroceries, 10_000, db.TxDirectionExpense, jul(3))
	addTx(t, q, userID, transportID, 4_000, db.TxDirectionExpense, jul(6))

	h := queries.NewTotalsHandler(q, testdb.Clock{At: jul(20)})

	all, err := h.Handle(context.Background(), queries.TransactionTotals{
		UserID: userID, From: rfc(jul(1)), To: rfc(jul(31)),
	})
	if err != nil {
		t.Fatalf("totals (all): %v", err)
	}

	if all.SpentMinor != 14_000 {
		t.Errorf("spent (all) = %d, want 14000", all.SpentMinor)
	}

	only, err := h.Handle(context.Background(), queries.TransactionTotals{
		UserID: userID, From: rfc(jul(1)), To: rfc(jul(31)), Category: transportID,
	})
	if err != nil {
		t.Fatalf("totals (by category): %v", err)
	}

	if only.SpentMinor != 4_000 {
		t.Errorf("spent (transport) = %d, want 4000", only.SpentMinor)
	}

	if only.IncomeMinor != 0 {
		t.Errorf("income (transport) = %d, want 0", only.IncomeMinor)
	}
}

// TestTotalsRangeIsHalfOpen — граница to исключающая (spent_at < to), как и у
// остальных агрегатов диапазона; трата ровно на to не попадает.
func TestTotalsRangeIsHalfOpen(t *testing.T) {
	pool, q := testdb.Pool(t)
	userID := testdb.NewUser(t, pool, q, jul(1), 1)

	addTx(t, q, userID, presetGroceries, 7_000, db.TxDirectionExpense, jul(10))

	h := queries.NewTotalsHandler(q, testdb.Clock{At: jul(20)})

	got, err := h.Handle(context.Background(), queries.TransactionTotals{
		UserID: userID,
		From:   rfc(jul(1)),
		To:     rfc(jul(10)), // ровно момент траты — снаружи полуинтервала
	})
	if err != nil {
		t.Fatalf("totals: %v", err)
	}

	if got.SpentMinor != 0 {
		t.Errorf("spent = %d, want 0 (трата на границе to не входит)", got.SpentMinor)
	}
}
