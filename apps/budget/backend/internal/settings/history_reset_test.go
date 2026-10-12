package settings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	balancecommands "budget_me/backend/internal/balance/commands"
	balancequeries "budget_me/backend/internal/balance/queries"
	"budget_me/backend/internal/settings/commands"
	"budget_me/backend/internal/testdb"
)

// Смена первого дня периода — единственная настройка, которая не меняется
// молча: она перекраивает все периоды, что приложение когда-либо показывало.
// Проверяем против настоящей БД — согласие пишется одним UPDATE вместе с
// обнулением якоря, и разъехаться они не должны.

func day(d int) time.Time { return time.Date(2026, 8, d, 9, 0, 0, 0, time.UTC) }

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

// TestChangingStartDayNeedsConfirmation — без явного согласия смена дня
// отклоняется, и НИЧЕГО не записывается: ни день, ни сброс.
func TestChangingStartDayNeedsConfirmation(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	update := commands.NewUpdateHandler(q, testdb.Clock{At: day(8)})

	_, err := update.Handle(ctx, commands.UpdateSettings{
		UserID:        userID,
		MonthStartDay: ptrInt(5),
	})
	if !errors.Is(err, commands.ErrHistoryResetRequired) {
		t.Fatalf("ошибка = %v, ожидалась ErrHistoryResetRequired", err)
	}

	got, err := commands.NewUpdateHandler(q, testdb.Clock{At: day(8)}).
		Handle(ctx, commands.UpdateSettings{UserID: userID})
	if err != nil {
		t.Fatalf("re-read settings: %v", err)
	}

	if got.MonthStartDay != 1 {
		t.Errorf("день начала = %d, ожидался 1 — отказ не должен был ничего сохранить", got.MonthStartDay)
	}

	if got.HistoryResetAt != nil {
		t.Error("отсечение проставлено, хотя смену отклонили")
	}
}

// TestConfirmedChangeCutsHistoryAndDropsAnchor — согласие двигает день,
// ставит отсечение и снимает якорь баланса: заявленное «у меня сейчас X»
// относилось к старой нарезке, поэтому приложение спросит остаток заново.
func TestConfirmedChangeCutsHistoryAndDropsAnchor(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	setBalance := balancecommands.NewSetHandler(q, testdb.Clock{At: day(2)})
	if _, err := setBalance.Handle(ctx, balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 1_000_000,
	}); err != nil {
		t.Fatalf("set anchor: %v", err)
	}

	at := day(8)

	got, err := commands.NewUpdateHandler(q, testdb.Clock{At: at}).Handle(ctx, commands.UpdateSettings{
		UserID:        userID,
		MonthStartDay: ptrInt(5),
		ResetHistory:  true,
	})
	if err != nil {
		t.Fatalf("confirmed update: %v", err)
	}

	if got.MonthStartDay != 5 {
		t.Errorf("день начала = %d, ожидался 5", got.MonthStartDay)
	}

	if got.HistoryResetAt == nil {
		t.Fatal("отсечение не проставлено после подтверждённой смены")
	}

	if !got.HistoryResetAt.Equal(at.UTC()) {
		t.Errorf("отсечение = %v, ожидалось %v", got.HistoryResetAt, at.UTC())
	}

	balance, err := balancequeries.NewGetHandler(q).
		Handle(ctx, balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if balance.Set {
		t.Error("якорь баланса уцелел — после сброса остаток должен спрашиваться заново")
	}
}

// TestSameDayIsNotAChange — повторная отправка ТОГО ЖЕ дня (так делает форма,
// сохраняя валюту) не должна ни требовать подтверждения, ни стоить пользователю
// якоря баланса.
func TestSameDayIsNotAChange(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)

	setBalance := balancecommands.NewSetHandler(q, testdb.Clock{At: day(2)})
	if _, err := setBalance.Handle(ctx, balancecommands.SetBalance{
		UserID:       userID,
		BalanceMinor: 1_000_000,
	}); err != nil {
		t.Fatalf("set anchor: %v", err)
	}

	got, err := commands.NewUpdateHandler(q, testdb.Clock{At: day(8)}).Handle(ctx, commands.UpdateSettings{
		UserID:        userID,
		MonthStartDay: ptrInt(1),
		Currency:      ptrStr("USD"),
	})
	if err != nil {
		t.Fatalf("currency-only update: %v", err)
	}

	if got.Currency != "USD" {
		t.Errorf("валюта = %q, ожидалась USD", got.Currency)
	}

	if got.HistoryResetAt != nil {
		t.Error("отсечение проставлено, хотя день не менялся")
	}

	balance, err := balancequeries.NewGetHandler(q).
		Handle(ctx, balancequeries.GetBalance{UserID: userID})
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if !balance.Set || balance.BalanceMinor != 1_000_000 {
		t.Errorf("баланс = %d (задан: %v), ожидался 1000000 — смена валюты не сбрасывает остаток",
			balance.BalanceMinor, balance.Set)
	}
}

// TestResetSurvivesLaterSettingsEdits — отсечение не должно теряться при
// последующих правках: сменил валюту после сброса — сброс остаётся в силе.
func TestResetSurvivesLaterSettingsEdits(t *testing.T) {
	pool, q := testdb.Pool(t)

	ctx := context.Background()
	userID := testdb.NewUser(t, pool, q, day(1), 1)
	at := day(8)

	if _, err := commands.NewUpdateHandler(q, testdb.Clock{At: at}).Handle(ctx, commands.UpdateSettings{
		UserID:        userID,
		MonthStartDay: ptrInt(5),
		ResetHistory:  true,
	}); err != nil {
		t.Fatalf("confirmed update: %v", err)
	}

	got, err := commands.NewUpdateHandler(q, testdb.Clock{At: day(9)}).Handle(ctx, commands.UpdateSettings{
		UserID:   userID,
		Currency: ptrStr("EUR"),
	})
	if err != nil {
		t.Fatalf("later currency edit: %v", err)
	}

	if got.HistoryResetAt == nil {
		t.Fatal("отсечение пропало после правки валюты")
	}

	if !got.HistoryResetAt.Equal(at.UTC()) {
		t.Errorf("отсечение = %v, ожидалось %v — правка валюты не должна его двигать", got.HistoryResetAt, at.UTC())
	}
}
