package balance_test

import (
	// Стандартная библиотека.
	"context"
	"os"
	"testing"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	// Внутренние пакеты.
	balancecommands "budget_me/backend/internal/balance/commands"
	balancequeries "budget_me/backend/internal/balance/queries"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	txcommands "budget_me/backend/internal/transactions/commands"
)

// Требует поднятую БД (DATABASE_URL) — единственный способ проверить это
// поведение честно: SumSignedSince — реальный SQL, а не то, что можно
// замокать интерфейсом (правило проекта — sqlc IS the repository).
//
// salaryCategoryID — сид из 0004_seed_categories.sql, общая income-категория:
// видна любому пользователю без привязки user_id.
const salaryCategoryID = "00000000-0000-4000-8000-000000000007"

// fakeClock даёт тесту управлять «сейчас» — так транзакцию можно проставить
// прошлым месяцем, а баланс перечитать как будто наступил следующий, не
// дожидаясь реального конца месяца.
type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

func newTestPool(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping DB-backed test")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool, db.New(pool)
}

func newTestUser(t *testing.T, pool *pgxpool.Pool, q *db.Queries, now time.Time) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	userID := ids.New()

	_, err := q.InsertUser(ctx, db.InsertUserParams{
		ID:            ids.ToPg(userID),
		FirstName:     "Carryover Test",
		MonthStartDay: 1,
		Currency:      db.CurrencyCodeBYN,
		Timezone:      "UTC",
		CreatedAt:     db.Time(now),
		UpdatedAt:     db.Time(now),
	})
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})

	return userID
}

func logIncome(t *testing.T, q *db.Queries, userID uuid.UUID, at time.Time, amountMinor int64, title string) {
	t.Helper()

	h := txcommands.NewCreateHandler(q, fakeClock{now: at})

	_, err := h.Handle(context.Background(), txcommands.CreateTransaction{
		UserID:      userID,
		CategoryID:  uuid.MustParse(salaryCategoryID),
		AmountMinor: amountMinor,
		Title:       title,
		SpentAt:     at,
		Direction:   db.TxDirectionIncome,
	})
	if err != nil {
		t.Fatalf("log income %q: %v", title, err)
	}
}

// TestBalanceCarriesAcrossMonthBoundary — ядро жалобы: доход, залогированный
// в одном месяце, обязан остаться в Остатке и после того, как календарь
// перевалил на следующий. SumSignedSince не знает о месяцах вообще (WHERE —
// только user_id и created_at >= anchor), поэтому граница месяца не должна на
// баланс никак влиять.
func TestBalanceCarriesAcrossMonthBoundary(t *testing.T) {
	pool, q := newTestPool(t)

	june := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	july := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	august := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	userID := newTestUser(t, pool, q, june)

	setHandler := balancecommands.NewSetHandler(q, fakeClock{now: june})
	if _, err := setHandler.Handle(context.Background(), balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 1_000_000, // 10 000.00 Br
	}); err != nil {
		t.Fatalf("set anchor: %v", err)
	}

	logIncome(t, q, userID, july, 300_000, "July salary")
	logIncome(t, q, userID, august, 50_000, "August top-up")

	getHandler := balancequeries.NewGetHandler(q)

	got, err := getHandler.Handle(context.Background(), balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	want := int64(1_000_000 + 300_000 + 50_000)

	if !got.Set {
		t.Fatalf("balance.Set = false, want true")
	}

	if got.BalanceMinor != want {
		t.Errorf("BalanceMinor = %d, want %d — July income did not carry into August", got.BalanceMinor, want)
	}
}

// TestBalanceUnsetWithoutAnchor — до первого ввода остатка баланс не
// придуманный ноль, а явное «не задан» (Set=false): транзакции одни не знают,
// что уже лежало в кошельке.
func TestBalanceUnsetWithoutAnchor(t *testing.T) {
	pool, q := newTestPool(t)

	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	userID := newTestUser(t, pool, q, now)

	getHandler := balancequeries.NewGetHandler(q)

	got, err := getHandler.Handle(context.Background(), balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if got.Set {
		t.Errorf("balance.Set = true, want false — no anchor was ever stated")
	}
}

// TestReanchorIgnoresOlderTransactions — переустановка остатка двигает якорь
// на «сейчас»: заявленная цифра — это правда на этот момент, поэтому всё, что
// было залогировано ДО неё, больше баланс не двигает (иначе исправление
// разъехалось бы с реальностью).
func TestReanchorIgnoresOlderTransactions(t *testing.T) {
	pool, q := newTestPool(t)

	july := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	august := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	userID := newTestUser(t, pool, q, july)

	setHandler := balancecommands.NewSetHandler(q, fakeClock{now: july})
	if _, err := setHandler.Handle(context.Background(), balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 1_000_000,
	}); err != nil {
		t.Fatalf("set first anchor: %v", err)
	}

	logIncome(t, q, userID, july, 300_000, "July salary")

	// Пользователь поправляет остаток в августе — новый якорь обнуляет базу.
	reanchor := balancecommands.NewSetHandler(q, fakeClock{now: august})
	if _, err := reanchor.Handle(context.Background(), balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 500_000,
	}); err != nil {
		t.Fatalf("re-anchor: %v", err)
	}

	getHandler := balancequeries.NewGetHandler(q)

	got, err := getHandler.Handle(context.Background(), balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if got.BalanceMinor != 500_000 {
		t.Errorf("BalanceMinor = %d, want 500000 — the July income should no longer count", got.BalanceMinor)
	}
}
