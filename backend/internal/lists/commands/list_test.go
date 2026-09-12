package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	commands "budget_me/backend/internal/lists/commands"
	"budget_me/backend/internal/lists/queries"
	"budget_me/backend/internal/testdb"
)

func day(d int) time.Time { return time.Date(2026, 9, d, 9, 0, 0, 0, time.UTC) }

const presetMedications = "00000000-0000-4000-9000-000000000001"

// TestCreateListRejectsBlankName — пробел не считается названием.
func TestCreateListRejectsBlankName(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	_, err := create.Handle(ctx, commands.CreateList{
		UserID: userID, Name: "   ", Color: "#ea6a6a",
	})

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.Invalid {
		t.Fatalf("ошибка = %v, ожидалась apperr.Invalid", err)
	}
}

// TestDeleteListCannotTouchPreset — WHERE user_id = свой никогда не заденет
// общий пресет (user_id IS NULL): удаление молча ничего не находит, и пресет
// остаётся виден и юзеру, и всем остальным.
func TestDeleteListCannotTouchPreset(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	remove := commands.NewDeleteHandler(q)
	list := queries.NewListHandler(q)

	if err := remove.Handle(ctx, commands.DeleteList{
		UserID: userID, ID: uuid.MustParse(presetMedications),
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rows, err := list.Handle(ctx, queries.ListLists{UserID: userID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	found := false

	for _, r := range rows {
		if r.ID.Valid && uuid.UUID(r.ID.Bytes).String() == presetMedications {
			found = true
		}
	}

	if !found {
		t.Error("пресет «Аптечка» пропал после попытки удалить его чужим user_id")
	}
}
