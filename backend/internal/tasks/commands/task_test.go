package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/ids"
	commands "budget_me/backend/internal/tasks/commands"
	"budget_me/backend/internal/testdb"
)

func day(d int) time.Time { return time.Date(2026, 9, d, 9, 0, 0, 0, time.UTC) }

// TestCreateTaskRejectsBlankTitle — пробел не считается заголовком.
func TestCreateTaskRejectsBlankTitle(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	_, err := create.Handle(ctx, commands.CreateTask{
		UserID: userID, Title: "   ", PlannedFor: "2026-09-01",
	})

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.Invalid {
		t.Fatalf("ошибка = %v, ожидалась apperr.Invalid", err)
	}
}

// TestSetTaskDoneUntick — отметка снимается обратно nil-ом, а не вторым
// значением; проверяем, что done_on реально возвращается в NULL.
func TestSetTaskDoneUntick(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	clk := testdb.Clock{At: day(5)}
	create := commands.NewCreateHandler(q, clk)
	setDone := commands.NewSetDoneHandler(q, clk)

	task, err := create.Handle(ctx, commands.CreateTask{
		UserID: userID, Title: "Позвонить маме", PlannedFor: "2026-09-05",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	taskID := ids.FromPg(task.ID)
	done := "2026-09-05"

	ticked, err := setDone.Handle(ctx, commands.SetTaskDone{
		UserID: userID, ID: taskID, DoneOn: &done,
	})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}

	if !ticked.DoneOn.Valid {
		t.Fatal("done_on не проставился")
	}

	unticked, err := setDone.Handle(ctx, commands.SetTaskDone{
		UserID: userID, ID: taskID, DoneOn: nil,
	})
	if err != nil {
		t.Fatalf("untick: %v", err)
	}

	if unticked.DoneOn.Valid {
		t.Error("done_on должен был вернуться в NULL")
	}
}

// TestUpdateTaskNotFound — редактирование чужой/несуществующей задачи — 404,
// а не паника и не тихий успех.
func TestUpdateTaskNotFound(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	update := commands.NewUpdateHandler(q, testdb.Clock{At: day(1)})

	_, err := update.Handle(ctx, commands.UpdateTask{
		UserID:     userID,
		ID:         ids.New(),
		Title:      "не существует",
		PlannedFor: "2026-09-01",
	})
	if !errors.Is(err, commands.ErrNotFound) {
		t.Fatalf("ошибка = %v, ожидалась ErrNotFound", err)
	}
}
