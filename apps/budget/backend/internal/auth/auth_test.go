package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ident "budget_me/account"
	"budget_me/account/accounttest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	accountcommands "budget_me/backend/internal/account/commands"
	"budget_me/backend/internal/accountlink"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/config"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/server"
	"budget_me/backend/internal/testdb"
)

const (
	authSecret = "auth-integration-test-secret-long-enough"
	botToken   = "12345:AUTH-TEST-BOT"
	grace      = 30 * 24 * time.Hour
)

type env struct {
	e        *echo.Echo
	q        *db.Queries
	pool     *pgxpool.Pool
	accounts *ident.Store
	accPool  *pgxpool.Pool
}

func setup(t *testing.T) env {
	t.Helper()

	pool, q := testdb.Pool(t)
	accPool := testdb.AccountPool(t)
	cfg := config.Config{AuthSecret: authSecret, TelegramBotToken: botToken, AccountDeletionGrace: grace}

	return env{
		e:        server.New(cfg, pool, accPool),
		q:        q,
		pool:     pool,
		accounts: ident.New(accPool),
		accPool:  accPool,
	}
}

func call(t *testing.T, e *echo.Echo, method, path, bearer string, body any) (int, map[string]any) {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("%s %s: %s", method, path, rec.Body.String())
	}

	return rec.Code, response
}

func data(t *testing.T, response map[string]any) map[string]any {
	t.Helper()

	d, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected response: %v", response)
	}

	return d
}

// telegramUser is a Telegram user with an id no other test uses.
func telegramUser() ident.TelegramUser {
	return ident.TelegramUser{ID: time.Now().UnixNano() % 1_000_000_000_000, FirstName: "Ann", Username: "ann"}
}

func signInTelegram(t *testing.T, v env, u ident.TelegramUser) (token string, userID string) {
	t.Helper()

	code, body := call(t, v.e, "POST", "/api/v1/auth/telegram", "", map[string]string{
		"init_data": accounttest.InitData(botToken, u, time.Now()),
	})
	if code != http.StatusOK {
		t.Fatalf("telegram sign-in: %d %v", code, body)
	}

	d := data(t, body)

	return d["token"].(string), d["user"].(map[string]any)["id"].(string)
}

// cleanup removes what a test created in both databases.
func cleanup(t *testing.T, v env, tgID int64) {
	t.Helper()

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = v.pool.Exec(ctx, "DELETE FROM users WHERE telegram_id = $1", tgID)
		_, _ = v.accPool.Exec(ctx, "DELETE FROM accounts WHERE telegram_id = $1", tgID)
	})
}

func TestTelegramSignInCreatesTheAccountAndThisAppsUser(t *testing.T) {
	v := setup(t)
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	token, userID := signInTelegram(t, v, tg)

	code, body := call(t, v.e, "GET", "/api/v1/me", token, nil)
	if code != http.StatusOK {
		t.Fatalf("me: %d %v", code, body)
	}

	me := data(t, body)
	if me["id"] != userID || me["username"] != "ann" || me["telegram_id"].(float64) != float64(tg.ID) {
		t.Fatalf("me = %v", me)
	}

	if _, pending := me["delete_at"]; pending {
		t.Fatalf("nothing is pending, but delete_at = %v", me["delete_at"])
	}

	user, err := v.q.GetUserByTelegramID(context.Background(), db.Int8(&tg.ID))
	if err != nil || !user.AccountID.Valid || ids.FromPg(user.ID).String() != userID {
		t.Fatalf("local user = %+v, %v", user, err)
	}

	// Signing in again reuses both.
	_, again := signInTelegram(t, v, tg)
	if again != userID {
		t.Fatalf("second sign-in made another user: %s vs %s", again, userID)
	}
}

func TestSignInKeepsAUserThatPredatesAccountsAndItsEmailLogin(t *testing.T) {
	v := setup(t)
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	email := "legacy-" + uuid.NewString() + "@example.com"
	hash, _ := ident.HashPassword("legacy-password")
	legacy := ids.New()
	now := db.Time(time.Now())

	if _, err := v.q.InsertUser(context.Background(), db.InsertUserParams{
		ID: ids.ToPg(legacy), TelegramID: db.Int8(&tg.ID), Email: db.Text(&email),
		PasswordHash: db.Text(&hash), FirstName: "Old", MonthStartDay: 1, Currency: db.CurrencyCodeBYN, Timezone: "UTC",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	_, userID := signInTelegram(t, v, tg)
	if userID != legacy.String() {
		t.Fatalf("the old user should stay the user: got %s, want %s", userID, legacy)
	}

	code, body := call(t, v.e, "POST", "/api/v1/auth/login", "", map[string]string{"email": email, "password": "legacy-password"})
	if code != http.StatusOK || data(t, body)["user"].(map[string]any)["id"] != legacy.String() {
		t.Fatalf("email login after the move: %d %v", code, body)
	}
}

func TestAPersonFromAnotherAppGetsTheSameAccount(t *testing.T) {
	v := setup(t)
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	// The other app signed this person in first and they linked an email there.
	other, err := v.accounts.SignInTelegram(context.Background(), tg, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	email := "shared-" + uuid.NewString() + "@example.com"
	if _, err := v.accounts.LinkEmail(context.Background(), other.ID, email, "shared-password", time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := v.accounts.Register(context.Background(), other.ID, ident.Learning); err != nil {
		t.Fatal(err)
	}

	token, _ := signInTelegram(t, v, tg)

	code, body := call(t, v.e, "GET", "/api/v1/me", token, nil)
	if code != http.StatusOK || data(t, body)["email"] != email {
		t.Fatalf("the email linked in the other app should show here: %d %v", code, body)
	}

	user, err := v.q.GetUserByTelegramID(context.Background(), db.Int8(&tg.ID))
	if err != nil || ids.FromPg(user.AccountID) != other.ID {
		t.Fatalf("user should hang off the shared account %s, got %+v (%v)", other.ID, user, err)
	}
}

func TestLinkEmailThroughTheAPI(t *testing.T) {
	v := setup(t)
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	token, _ := signInTelegram(t, v, tg)
	email := "link-" + uuid.NewString() + "@example.com"

	code, body := call(t, v.e, "POST", "/api/v1/auth/link/email", token, map[string]string{"email": email, "password": "long-enough-password"})
	if code != http.StatusOK || data(t, body)["email"] != email {
		t.Fatalf("link: %d %v", code, body)
	}

	code, body = call(t, v.e, "POST", "/api/v1/auth/login", "", map[string]string{"email": email, "password": "long-enough-password"})
	if code != http.StatusOK {
		t.Fatalf("login with the linked email: %d %v", code, body)
	}

	// Someone else cannot take the address.
	other := telegramUser()
	other.ID++
	cleanup(t, v, other.ID)

	otherToken, _ := signInTelegram(t, v, other)

	code, _ = call(t, v.e, "POST", "/api/v1/auth/link/email", otherToken, map[string]string{"email": email, "password": "another-long-password"})
	if code != http.StatusConflict {
		t.Fatalf("taken email: %d, want 409", code)
	}
}

func TestDeletionRequestShowsADeadlineAndCanBeCancelled(t *testing.T) {
	v := setup(t)
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	token, _ := signInTelegram(t, v, tg)

	code, body := call(t, v.e, "POST", "/api/v1/account/deletion", token, nil)
	if code != http.StatusOK {
		t.Fatalf("request: %d %v", code, body)
	}

	requested, err := time.Parse(time.RFC3339, data(t, body)["requestedAt"].(string))
	if err != nil {
		t.Fatal(err)
	}

	deleteAt, err := time.Parse(time.RFC3339, data(t, body)["deleteAt"].(string))
	if err != nil || deleteAt.Sub(requested) != grace {
		t.Fatalf("deadline = %v after %v (%v)", deleteAt, requested, err)
	}

	_, body = call(t, v.e, "GET", "/api/v1/me", token, nil)
	if data(t, body)["delete_at"] == nil {
		t.Fatalf("/me should report the pending deletion: %v", body)
	}

	// The nightly advice pass reads this app's own flag, so it follows along.
	local, err := v.q.GetUserByTelegramID(context.Background(), db.Int8(&tg.ID))
	if err != nil || !local.DeletionRequestedAt.Valid {
		t.Fatalf("local deletion flag not set: %+v, %v", local, err)
	}

	code, body = call(t, v.e, "DELETE", "/api/v1/account/deletion", token, nil)
	if code != http.StatusOK || data(t, body)["deleteAt"] != nil {
		t.Fatalf("cancel: %d %v", code, body)
	}

	_, body = call(t, v.e, "GET", "/api/v1/me", token, nil)
	if _, pending := data(t, body)["delete_at"]; pending {
		t.Fatalf("cancelled, but /me still reports %v", data(t, body)["delete_at"])
	}

	local, err = v.q.GetUserByTelegramID(context.Background(), db.Int8(&tg.ID))
	if err != nil || local.DeletionRequestedAt.Valid {
		t.Fatalf("local deletion flag still set: %+v, %v", local, err)
	}
}

func TestPurgeErasesThisAppsDataAndWaitsForTheOtherApp(t *testing.T) {
	v := setup(t)
	ctx := context.Background()
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	token, userID := signInTelegram(t, v, tg)

	// A second app also holds data for the account.
	user, err := v.q.GetUserByTelegramID(ctx, db.Int8(&tg.ID))
	if err != nil {
		t.Fatal(err)
	}

	accountID := ids.FromPg(user.AccountID)
	if err := v.accounts.Register(ctx, accountID, ident.Learning); err != nil {
		t.Fatal(err)
	}

	if code, body := call(t, v.e, "POST", "/api/v1/account/deletion", token, nil); code != http.StatusOK {
		t.Fatalf("request: %d %v", code, body)
	}

	purge := func(at time.Time) {
		t.Helper()

		h := accountcommands.NewPurgeHandler(v.q, v.accounts, testdb.Clock{At: at}, grace)
		if _, err := h.Handle(ctx); err != nil {
			t.Fatal(err)
		}
	}

	// Still inside the grace period: nothing is touched.
	purge(time.Now().Add(grace - time.Hour))

	if _, err := v.q.GetUserByID(ctx, ids.ToPg(uuid.MustParse(userID))); err != nil {
		t.Fatalf("user vanished inside the grace period: %v", err)
	}

	// Past it: this app's user is gone, the account stays for the other app.
	purge(time.Now().Add(grace + time.Hour))

	if _, err := v.q.GetUserByID(ctx, ids.ToPg(uuid.MustParse(userID))); !db.IsNotFound(err) {
		t.Fatalf("user should be erased, got %v", err)
	}

	if _, err := v.accounts.ByID(ctx, accountID); err != nil {
		t.Fatalf("the account must outlive this app while learning still holds it: %v", err)
	}
}

func TestBackfillLinksUsersThatPredateAccountsOnce(t *testing.T) {
	v := setup(t)
	ctx := context.Background()
	tg := telegramUser()
	cleanup(t, v, tg.ID)

	legacy := ids.New()
	now := db.Time(time.Now())

	if _, err := v.q.InsertUser(ctx, db.InsertUserParams{
		ID: ids.ToPg(legacy), TelegramID: db.Int8(&tg.ID), FirstName: "Old", MonthStartDay: 1, Currency: db.CurrencyCodeBYN, Timezone: "UTC",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	links := accountlink.New(v.q, v.accounts, clock.New())

	if _, err := links.Backfill(ctx); err != nil {
		t.Fatal(err)
	}

	user, err := v.q.GetUserByID(ctx, ids.ToPg(legacy))
	if err != nil || !user.AccountID.Valid {
		t.Fatalf("not linked after backfill: %+v, %v", user, err)
	}

	if _, err := v.accounts.ByID(ctx, ids.FromPg(user.AccountID)); err != nil {
		t.Fatalf("account missing after backfill: %v", err)
	}

	again, err := links.Backfill(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Other tests' leftovers are not ours to count; this user must not be re-linked.
	after, _ := v.q.GetUserByID(ctx, ids.ToPg(legacy))
	if after.AccountID != user.AccountID {
		t.Fatalf("second backfill moved the user: %v → %v (relinked %d)", user.AccountID, after.AccountID, again)
	}
}
