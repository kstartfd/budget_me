package commands_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	commands "budget_me/backend/internal/items/commands"
	"budget_me/backend/internal/photostore"
	"budget_me/backend/internal/testdb"
)

// tinyPNG is a minimal valid image for photostore.Save to decode — the point
// of these tests is the upload/replace/cleanup plumbing, not image fidelity.
func tinyPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{R: 200, G: 40, B: 40, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode fixture png: %v", err)
	}

	return buf.Bytes()
}

func createItem(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID) db.Item {
	t.Helper()

	create := commands.NewCreateHandler(q, testdb.Clock{At: day(1)})

	item, err := create.Handle(ctx, commands.CreateItem{
		UserID: userID,
		ListID: uuid.MustParse(presetMedications),
		Title:  "Крем",
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	return item
}

// TestUploadItemPhotoStoresAndReplaces — the first upload writes a file and
// stamps its path onto the item; a second upload swaps in a new file and
// removes the old one, so no orphan is left behind.
func TestUploadItemPhotoStoresAndReplaces(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	item := createItem(t, ctx, q, userID)

	dir := t.TempDir()
	upload := commands.NewUploadPhotoHandler(q, testdb.Clock{At: day(1)}, photostore.New(dir))

	first, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID), Data: tinyPNG(t),
	})
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	firstPath := db.TextPtr(first.PhotoPath)
	if firstPath == nil {
		t.Fatal("photo_path is nil after upload")
	}

	if _, err := os.Stat(filepath.Join(dir, *firstPath)); err != nil {
		t.Fatalf("stat saved file: %v", err)
	}

	second, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID), Data: tinyPNG(t),
	})
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}

	secondPath := db.TextPtr(second.PhotoPath)
	if secondPath == nil || *secondPath == *firstPath {
		t.Fatalf("second path = %v, expected a new file", secondPath)
	}

	if _, err := os.Stat(filepath.Join(dir, *firstPath)); !os.IsNotExist(err) {
		t.Error("old photo file should have been removed after replace")
	}
}

// TestUploadItemPhotoRejectsGarbage — bytes that are not a decodable image
// never reach disk and the item is left untouched.
func TestUploadItemPhotoRejectsGarbage(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	item := createItem(t, ctx, q, userID)

	upload := commands.NewUploadPhotoHandler(q, testdb.Clock{At: day(1)}, photostore.New(t.TempDir()))

	_, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID), Data: []byte("not an image"),
	})
	if !errors.Is(err, commands.ErrInvalidPhoto) {
		t.Fatalf("ошибка = %v, ожидалась ErrInvalidPhoto", err)
	}
}

// TestUploadItemPhotoRejectsOtherUsersItem — mirrors items' own ownership
// pattern: an item id that isn't the caller's is treated as not found.
func TestUploadItemPhotoRejectsOtherUsersItem(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	owner := testdb.NewUser(t, pool, q, day(1), 1)
	intruder := testdb.NewUser(t, pool, q, day(1), 1)
	item := createItem(t, ctx, q, owner)

	upload := commands.NewUploadPhotoHandler(q, testdb.Clock{At: day(1)}, photostore.New(t.TempDir()))

	_, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: intruder, ID: ids.FromPg(item.ID), Data: tinyPNG(t),
	})
	if !errors.Is(err, commands.ErrNotFound) {
		t.Fatalf("ошибка = %v, ожидалась ErrNotFound", err)
	}
}

// TestRemoveItemPhotoDeletesFile — clearing the photo nulls the column AND
// removes the file from disk, not just one or the other.
func TestRemoveItemPhotoDeletesFile(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	item := createItem(t, ctx, q, userID)

	dir := t.TempDir()
	store := photostore.New(dir)
	upload := commands.NewUploadPhotoHandler(q, testdb.Clock{At: day(1)}, store)
	remove := commands.NewRemovePhotoHandler(q, testdb.Clock{At: day(1)}, store)

	uploaded, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID), Data: tinyPNG(t),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	path := db.TextPtr(uploaded.PhotoPath)

	cleared, err := remove.Handle(ctx, commands.RemoveItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID),
	})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	if db.TextPtr(cleared.PhotoPath) != nil {
		t.Error("photo_path should be nil after remove")
	}

	if _, err := os.Stat(filepath.Join(dir, *path)); !os.IsNotExist(err) {
		t.Error("photo file should have been deleted from disk")
	}
}

// TestDeleteItemRemovesPhotoFile — deleting the item itself must not leave an
// orphaned photo behind either.
func TestDeleteItemRemovesPhotoFile(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	item := createItem(t, ctx, q, userID)

	dir := t.TempDir()
	store := photostore.New(dir)
	upload := commands.NewUploadPhotoHandler(q, testdb.Clock{At: day(1)}, store)
	remove := commands.NewDeleteHandler(q, store)

	uploaded, err := upload.Handle(ctx, commands.UploadItemPhoto{
		UserID: userID, ID: ids.FromPg(item.ID), Data: tinyPNG(t),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	path := db.TextPtr(uploaded.PhotoPath)

	if err := remove.Handle(ctx, commands.DeleteItem{UserID: userID, ID: ids.FromPg(item.ID)}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, *path)); !os.IsNotExist(err) {
		t.Error("photo file should have been deleted alongside the item")
	}
}
