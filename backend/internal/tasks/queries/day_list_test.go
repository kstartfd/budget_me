package queries_test

import (
	"context"
	"testing"
	"time"

	"budget_me/backend/internal/ids"
	commands "budget_me/backend/internal/tasks/commands"
	"budget_me/backend/internal/tasks/queries"
	"budget_me/backend/internal/testdb"
)

func day(d int) time.Time { return time.Date(2026, 9, d, 9, 0, 0, 0, time.UTC) }

func strPtr(s string) *string { return &s }

// TestDayListCarriesOverUndoneWork — сердце фичи: перенос НИЧЕГО не пишет,
// день — это запрос. Задача, запланированная три дня назад и не отмеченная,
// обязана прийти в список «сегодня»; задача, отмеченная вчера, — не обязана
// (она принадлежит вчерашнему дню, не сегодняшнему).
func TestDayListCarriesOverUndoneWork(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	clk := testdb.Clock{At: day(12)}
	create := commands.NewCreateHandler(q, clk)
	setDone := commands.NewSetDoneHandler(q, clk)
	list := queries.NewDayListHandler(q)

	// Просрочена на 3 дня, не отмечена — должна дожить до сегодня.
	overdue, err := create.Handle(ctx, commands.CreateTask{
		UserID: userID, Title: "Сдать анализы", PlannedFor: "2026-09-09",
	})
	if err != nil {
		t.Fatalf("create overdue: %v", err)
	}

	// Запланирована и выполнена вчера — принадлежит вчера, не сегодня.
	yesterday, err := create.Handle(ctx, commands.CreateTask{
		UserID: userID, Title: "Разобрать почту", PlannedFor: "2026-09-11",
	})
	if err != nil {
		t.Fatalf("create yesterday: %v", err)
	}

	if _, err := setDone.Handle(ctx, commands.SetTaskDone{
		UserID: userID, ID: ids.FromPg(yesterday.ID), DoneOn: strPtr("2026-09-11"),
	}); err != nil {
		t.Fatalf("mark done yesterday: %v", err)
	}

	// Запланирована на сегодня — обычный случай.
	today, err := create.Handle(ctx, commands.CreateTask{
		UserID: userID, Title: "Оплатить хостинг", PlannedFor: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("create today: %v", err)
	}

	rows, err := list.Handle(ctx, queries.DayTasks{UserID: userID, Day: "2026-09-12"})
	if err != nil {
		t.Fatalf("list day: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("на сегодня попало %d задач, ожидалось 2 (просроченная + сегодняшняя): %+v", len(rows), rows)
	}

	titles := map[string]bool{}
	for _, r := range rows {
		titles[r.Title] = true
	}

	if !titles[overdue.Title] {
		t.Error("просроченная задача не перенеслась на сегодня")
	}

	if !titles[today.Title] {
		t.Error("сегодняшняя задача не попала в список")
	}

	if titles[yesterday.Title] {
		t.Error("вчера выполненная задача не должна показываться сегодня")
	}
}
