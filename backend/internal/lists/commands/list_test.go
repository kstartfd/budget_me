package commands_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	commands "budget_me/backend/internal/lists/commands"
	"budget_me/backend/internal/lists/queries"
	"budget_me/backend/internal/photostore"
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

	remove := commands.NewDeleteHandler(pool, photostore.New(t.TempDir()))
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

func TestDeleteListRemovesItemsAndPhotos(t *testing.T) {
	pool, q := testdb.Pool(t)
	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})
	list, err := create.Handle(ctx, commands.CreateList{UserID: userID, Name: "Витамины", Color: "#ea6a6a"})
	if err != nil {
		t.Fatalf("create list: %v", err)
	}

	photosDir := t.TempDir()
	for i := range 2 {
		itemID := ids.New()
		_, err := q.InsertItem(ctx, db.InsertItemParams{
			ID: ids.ToPg(itemID), ListID: list.ID, UserID: ids.ToPg(userID),
			Title: "Вещь", Verdict: db.ItemVerdictUnknown,
			CreatedAt: db.Time(day(i + 1)),
		})
		if err != nil {
			t.Fatalf("insert item: %v", err)
		}
		photo := filepath.Join("items", itemID.String()+".jpg")
		if err := os.MkdirAll(filepath.Join(photosDir, "items"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(photosDir, photo), []byte("photo"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = q.UpdateItemPhoto(ctx, db.UpdateItemPhotoParams{
			ID: ids.ToPg(itemID), UserID: ids.ToPg(userID),
			PhotoPath: db.Text(&photo), UpdatedAt: db.Time(day(2)),
		})
		if err != nil {
			t.Fatalf("attach photo: %v", err)
		}
	}

	remove := commands.NewDeleteHandler(pool, photostore.New(photosDir))
	if err := remove.Handle(ctx, commands.DeleteList{UserID: userID, ID: uuid.UUID(list.ID.Bytes)}); err != nil {
		t.Fatalf("delete list: %v", err)
	}
	var remainingLists int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM item_lists WHERE id = $1", uuid.UUID(list.ID.Bytes)).Scan(&remainingLists); err != nil || remainingLists != 0 {
		t.Fatalf("remaining lists = %d, error = %v", remainingLists, err)
	}
	items, err := q.ListItemsForList(ctx, db.ListItemsForListParams{ListID: list.ID, UserID: ids.ToPg(userID)})
	if err != nil || len(items) != 0 {
		t.Fatalf("remaining items = %d, error = %v", len(items), err)
	}
	entries, err := os.ReadDir(filepath.Join(photosDir, "items"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("remaining photos = %d, error = %v", len(entries), err)
	}
	if err := remove.Handle(ctx, commands.DeleteList{UserID: userID, ID: uuid.UUID(list.ID.Bytes)}); err != nil {
		t.Fatalf("repeated delete: %v", err)
	}
}

// TestUpdateListRenamesPersonal — свой личный список можно переименовать и
// перекрасить.
func TestUpdateListRenamesPersonal(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})
	update := commands.NewUpdateHandler(q)

	list, err := create.Handle(ctx, commands.CreateList{
		UserID: userID, Name: "Витамины", Color: "#ea6a6a",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	renamed, err := update.Handle(ctx, commands.UpdateList{
		UserID: userID,
		ID:     uuid.UUID(list.ID.Bytes),
		Name:   "Добавки",
		Color:  "#7c86ff",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	name := db.TextPtr(renamed.Name)
	if name == nil || *name != "Добавки" || renamed.Color != "#7c86ff" {
		t.Errorf("name/color = %v/%q, ожидались Добавки/#7c86ff", name, renamed.Color)
	}
}

// TestUpdateListCannotTouchPreset — тот же WHERE user_id = свой, что и у
// удаления: попытка переименовать пресет чужим user_id ничего не находит.
func TestUpdateListCannotTouchPreset(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	update := commands.NewUpdateHandler(q)

	_, err := update.Handle(ctx, commands.UpdateList{
		UserID: userID,
		ID:     uuid.MustParse(presetMedications),
		Name:   "Захват",
		Color:  "#7c86ff",
	})
	if !errors.Is(err, commands.ErrNotFound) {
		t.Fatalf("ошибка = %v, ожидалась ErrNotFound", err)
	}
}

// TestUpdateListRejectsBlankName — то же правило, что и при создании.
func TestUpdateListRejectsBlankName(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})
	update := commands.NewUpdateHandler(q)

	list, err := create.Handle(ctx, commands.CreateList{
		UserID: userID, Name: "Витамины", Color: "#ea6a6a",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = update.Handle(ctx, commands.UpdateList{
		UserID: userID,
		ID:     uuid.UUID(list.ID.Bytes),
		Name:   "   ",
		Color:  "#ea6a6a",
	})

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.Invalid {
		t.Fatalf("ошибка = %v, ожидалась apperr.Invalid", err)
	}
}
