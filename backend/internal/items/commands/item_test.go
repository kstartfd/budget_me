package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	commands "budget_me/backend/internal/items/commands"
	"budget_me/backend/internal/testdb"
)

func day(d int) time.Time { return time.Date(2026, 9, d, 9, 0, 0, 0, time.UTC) }

const presetMedications = "00000000-0000-4000-9000-000000000001"

// insertPersonalList creates a personal (non-preset) list directly through
// sqlc, standing in for lists/commands.CreateHandler — this package tests
// items, not list creation, and pulling in the lists package just for a test
// fixture would be an odd cross-feature dependency.
func insertPersonalList(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID) uuid.UUID {
	t.Helper()

	listID := ids.New()

	_, err := q.InsertList(ctx, db.InsertListParams{
		ID:        ids.ToPg(listID),
		UserID:    ids.ToPg(userID),
		Name:      db.Text(strPtr("Личный список")),
		Color:     "#7c86ff",
		CreatedAt: db.Time(day(1)),
	})
	if err != nil {
		t.Fatalf("insert personal list: %v", err)
	}

	return listID
}

func strPtr(s string) *string { return &s }

// TestCreateItemInPreset — вещь в общем пресете («Аптечка») должна создаться:
// пресет виден всем, а не переданный вердикт по умолчанию становится unknown.
func TestCreateItemInPreset(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	rating := 8

	item, err := create.Handle(ctx, commands.CreateItem{
		UserID: userID,
		ListID: uuid.MustParse(presetMedications),
		Title:  "Нурофен",
		Rating: &rating,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if item.Verdict != db.ItemVerdictUnknown {
		t.Errorf("verdict = %q, ожидался unknown", item.Verdict)
	}
}

// TestCreateItemRejectsOtherUsersList — нельзя подсунуть чужой личный список,
// подобрав его uuid: он не входит в «общий ИЛИ свой» (ListVisibleToUser).
func TestCreateItemRejectsOtherUsersList(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	owner := testdb.NewUser(t, pool, q, day(1), 1)
	intruder := testdb.NewUser(t, pool, q, day(1), 1)

	ownersList := insertPersonalList(t, ctx, q, owner)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	_, err := create.Handle(ctx, commands.CreateItem{
		UserID: intruder,
		ListID: ownersList,
		Title:  "Чужая вещь",
	})
	if !errors.Is(err, commands.ErrInvalidList) {
		t.Fatalf("ошибка = %v, ожидалась ErrInvalidList", err)
	}
}

// TestCreateItemRejectsOutOfRangeRating — 0 и 11 отклоняются, диапазон 1..10.
func TestCreateItemRejectsOutOfRangeRating(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	for _, bad := range []int{0, 11} {
		rating := bad

		_, err := create.Handle(ctx, commands.CreateItem{
			UserID: userID,
			ListID: uuid.MustParse(presetMedications),
			Title:  "Тест",
			Rating: &rating,
		})
		if !errors.Is(err, commands.ErrInvalidRating) {
			t.Errorf("rating=%d: ошибка = %v, ожидалась ErrInvalidRating", bad, err)
		}
	}
}
