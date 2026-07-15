package queries

import (
	// Стандартная библиотека.
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// CategorySpend — одна строка «сколько потрачено по категории» за период. Это
// доменный результат запроса (уже развёрнутый из pgtype в обычные Go-типы), а НЕ
// строка БД db.SumSpentByCategoryInRangeRow и НЕ JSON-ответ — тот соберёт слой
// dto в родительском пакете. CategoryID — обычный uuid.UUID (NULL-бакет уже
// отфильтрован, см. toCategorySpends).
//
// PHP-аналогия: маленький readonly-DTO { UuidInterface $categoryId; int $spentMinor }.
type CategorySpend struct {
	CategoryID uuid.UUID
	SpentMinor int64
}

// resolveRange приводит опциональные границы к паре МОМЕНТОВ [from, to) — один
// раз на весь пакет, чтобы логика не расползалась по list/summary/spent/by_category.
//
// Границы приезжают как RFC3339 В UTC: «месяц» — понятие часового пояса, а пояс
// знает только клиент. Он и присылает готовые моменты (полночь 1 июля в Минске =
// 2026-06-30T21:00:00Z). Бэкенд живёт в чистом UTC и ничего не пересчитывает —
// так нечему рассинхронизироваться, и переезд юзера в другой пояс ничего не ломает.
//
// Правила: from пусто → начало текущего UTC-месяца, to пусто → «сейчас»; to обязано
// быть строго позже from, иначе apperr.NewInvalid → HTTP 400.
//
// PHP-аналогия — приватный хелпер, принимающий две ?string и отдающий пару
// DateTimeImmutable; ошибка здесь — обычное третье возвращаемое значение, без throw.
func resolveRange(clk clock.Clock, fromStr, toStr string) (time.Time, time.Time, error) {
	now := clk.Now()

	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	if fromStr != "" {
		p, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, apperr.NewInvalid("from must be RFC3339 (UTC)")
		}

		from = p
	}

	to := now

	if toStr != "" {
		p, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, apperr.NewInvalid("to must be RFC3339 (UTC)")
		}

		to = p
	}

	// to должно быть строго позже from — защита от перевёрнутых/битых границ.
	if !to.After(from) {
		return time.Time{}, time.Time{}, apperr.NewInvalid("invalid date range")
	}

	return from, to, nil
}

// toCategorySpends разворачивает строки sqlc в []CategorySpend, ПРОПУСКАЯ NULL-бакет
// (category_id IS NULL — след удалённой категории при ON DELETE SET NULL): фронт
// ждёт непустой categoryId. Общий хелпер для by_category и summary, чтобы фильтр и
// маппинг были написаны один раз. Возвращаемый срез всегда НЕ nil (make с cap),
// поэтому JSON даст [], а не null.
func toCategorySpends(rows []db.SumSpentByCategoryInRangeRow) []CategorySpend {
	out := make([]CategorySpend, 0, len(rows))

	for _, row := range rows {
		if !row.CategoryID.Valid {
			continue
		}

		out = append(out, CategorySpend{
			CategoryID: ids.FromPg(row.CategoryID),
			SpentMinor: row.SpentMinor,
		})
	}

	return out
}
