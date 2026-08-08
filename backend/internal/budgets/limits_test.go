package budgets_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/budgets/commands"
	"budget_me/backend/internal/budgets/queries"
	"budget_me/backend/internal/testdb"
)

// Лимиты живут в SQL: «действующий лимит месяца» — это DISTINCT ON с
// valid_from <= month, и проверить его честно можно только против настоящей
// Postgres (sqlc IS the repository, мокать нечего).

var groceries = uuid.MustParse(testdb.PresetExpenseCategory)

func march(day int) time.Time { return time.Date(2026, 3, day, 12, 0, 0, 0, time.UTC) }

// effectiveLimit — что экран лимитов покажет за указанный месяц.
func effectiveLimit(t *testing.T, h queries.ListHandler, userID uuid.UUID, month string) (int64, bool) {
	t.Helper()

	got, err := h.Handle(context.Background(), queries.ListBudgets{UserID: userID, Month: month})
	if err != nil {
		t.Fatalf("list budgets for %s: %v", month, err)
	}

	for _, b := range got {
		if b.CategoryID == groceries {
			return b.LimitMinor, true
		}
	}

	return 0, false
}

// TestLimitCarriesIntoLaterMonths — ради этого всё и затевалось: лимит,
// заданный один раз в марте, обязан действовать и в апреле, и в июле, без
// повторного ввода каждый месяц.
func TestLimitCarriesIntoLaterMonths(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)

	save := commands.NewSaveHandler(q, testdb.Clock{At: march(3)})
	if _, err := save.Handle(context.Background(), commands.SaveBudget{
		UserID:     userID,
		CategoryID: groceries,
		ValidFrom:  "2026-03",
		LimitMinor: 50_000,
	}); err != nil {
		t.Fatalf("save limit: %v", err)
	}

	list := queries.NewListHandler(q)

	for _, month := range []string{"2026-03", "2026-04", "2026-07"} {
		limit, ok := effectiveLimit(t, list, userID, month)
		if !ok {
			t.Errorf("%s: лимита нет, а он должен был перенестись с марта", month)

			continue
		}

		if limit != 50_000 {
			t.Errorf("%s: лимит = %d, ожидался 50000", month, limit)
		}
	}
}

// TestLimitDoesNotLeakIntoEarlierMonths — перенос смотрит только вперёд:
// в феврале лимита ещё не было, и мартовская запись его туда не добавляет.
func TestLimitDoesNotLeakIntoEarlierMonths(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)

	save := commands.NewSaveHandler(q, testdb.Clock{At: march(3)})
	if _, err := save.Handle(context.Background(), commands.SaveBudget{
		UserID:     userID,
		CategoryID: groceries,
		ValidFrom:  "2026-03",
		LimitMinor: 50_000,
	}); err != nil {
		t.Fatalf("save limit: %v", err)
	}

	if _, ok := effectiveLimit(t, queries.NewListHandler(q), userID, "2026-02"); ok {
		t.Error("февраль получил лимит, заданный только с марта")
	}
}

// TestNewerLimitReplacesOlderOne — история честная: смена лимита в июне не
// переписывает март, но с июня действует уже новая цифра.
func TestNewerLimitReplacesOlderOne(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)
	save := commands.NewSaveHandler(q, testdb.Clock{At: march(3)})
	ctx := context.Background()

	for _, step := range []struct {
		month string
		limit int64
	}{
		{"2026-03", 50_000},
		{"2026-06", 80_000},
	} {
		if _, err := save.Handle(ctx, commands.SaveBudget{
			UserID:     userID,
			CategoryID: groceries,
			ValidFrom:  step.month,
			LimitMinor: step.limit,
		}); err != nil {
			t.Fatalf("save limit for %s: %v", step.month, err)
		}
	}

	list := queries.NewListHandler(q)

	for _, want := range []struct {
		month string
		limit int64
	}{
		{"2026-03", 50_000},
		{"2026-05", 50_000},
		{"2026-06", 80_000},
		{"2026-09", 80_000},
	} {
		limit, ok := effectiveLimit(t, list, userID, want.month)
		if !ok {
			t.Errorf("%s: лимита нет", want.month)

			continue
		}

		if limit != want.limit {
			t.Errorf("%s: лимит = %d, ожидался %d", want.month, limit, want.limit)
		}
	}
}

// TestRemovedLimitStaysRemoved — надгробие: снять лимит в июне нельзя простым
// DELETE строки, иначе июнь унаследовал бы мартовскую цифру обратно. При этом
// март обязан остаться с лимитом — удаление сегодня не переписывает прошлое.
func TestRemovedLimitStaysRemoved(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)
	ctx := context.Background()

	save := commands.NewSaveHandler(q, testdb.Clock{At: march(3)})
	if _, err := save.Handle(ctx, commands.SaveBudget{
		UserID:     userID,
		CategoryID: groceries,
		ValidFrom:  "2026-03",
		LimitMinor: 50_000,
	}); err != nil {
		t.Fatalf("save limit: %v", err)
	}

	remove := commands.NewDeleteHandler(q, testdb.Clock{At: time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)})
	if err := remove.Handle(ctx, commands.DeleteBudget{
		UserID:     userID,
		CategoryID: groceries,
		Month:      "2026-06",
	}); err != nil {
		t.Fatalf("delete limit: %v", err)
	}

	list := queries.NewListHandler(q)

	if limit, ok := effectiveLimit(t, list, userID, "2026-03"); !ok || limit != 50_000 {
		t.Errorf("март: лимит = %d (есть: %v), ожидался 50000 — прошлое трогать нельзя", limit, ok)
	}

	for _, month := range []string{"2026-06", "2026-08"} {
		if _, ok := effectiveLimit(t, list, userID, month); ok {
			t.Errorf("%s: лимит вернулся, хотя его сняли с июня", month)
		}
	}
}

// TestRemovingFirstEverLimitLeavesNothing — если наследовать нечего, надгробие
// не нужно: строки просто уходят, и категория снова «без лимита».
func TestRemovingFirstEverLimitLeavesNothing(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)
	ctx := context.Background()

	save := commands.NewSaveHandler(q, testdb.Clock{At: march(3)})
	if _, err := save.Handle(ctx, commands.SaveBudget{
		UserID:     userID,
		CategoryID: groceries,
		ValidFrom:  "2026-03",
		LimitMinor: 50_000,
	}); err != nil {
		t.Fatalf("save limit: %v", err)
	}

	remove := commands.NewDeleteHandler(q, testdb.Clock{At: march(4)})
	if err := remove.Handle(ctx, commands.DeleteBudget{
		UserID:     userID,
		CategoryID: groceries,
		Month:      "2026-03",
	}); err != nil {
		t.Fatalf("delete limit: %v", err)
	}

	var rows int

	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM budgets WHERE user_id = $1 AND category_id = $2",
		userID, groceries,
	).Scan(&rows); err != nil {
		t.Fatalf("count budgets: %v", err)
	}

	if rows != 0 {
		t.Errorf("осталось строк: %d, ожидалось 0 — наследовать было нечего, надгробие лишнее", rows)
	}

	if _, ok := effectiveLimit(t, queries.NewListHandler(q), userID, "2026-03"); ok {
		t.Error("март всё ещё с лимитом после удаления")
	}
}

// TestDeleteIsIdempotent — фронт удаляет оптимистично; повторное удаление уже
// снятого лимита не должно быть ошибкой.
func TestDeleteIsIdempotent(t *testing.T) {
	pool, q := testdb.Pool(t)

	userID := testdb.NewUser(t, pool, q, march(1), 1)
	ctx := context.Background()
	remove := commands.NewDeleteHandler(q, testdb.Clock{At: march(4)})

	for i := range 2 {
		if err := remove.Handle(ctx, commands.DeleteBudget{
			UserID:     userID,
			CategoryID: groceries,
			Month:      "2026-03",
		}); err != nil {
			t.Fatalf("удаление #%d вернуло ошибку: %v", i+1, err)
		}
	}
}
