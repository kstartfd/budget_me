package commands

import (
	// Стандартная библиотека.
	"context"
	"strings"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// UpdateTransaction — команда: ЧТО и У КОГО меняем. Это CreateTransaction плюс ID
// уже существующей строки. created_at сознательно НЕ входит — правка не двигает
// момент создания (в PHP это был бы тот же readonly-DTO, но с id записи).
type UpdateTransaction struct {
	UserID      uuid.UUID
	ID          uuid.UUID
	CategoryID  uuid.UUID
	AmountMinor int64
	Title       string
	SpentAt     time.Time
	Direction   db.TxDirection
	Counted     bool
}

// UpdateHandler — обработчик команды. Часы не нужны (created_at не трогаем),
// поэтому держит только sqlc-запросы.
type UpdateHandler struct {
	q *db.Queries
}

func NewUpdateHandler(q *db.Queries) UpdateHandler {
	return UpdateHandler{q: q}
}

// Handle проверяет ровно те же инварианты, что и создание (title непустой,
// amount>0, категория видима и того же направления), затем обновляет строку.
// UpdateTransaction — `:one` с RETURNING: если строки с таким id У ЭТОГО юзера
// нет, sqlc вернёт «нет строк» → мапим в ErrNotFound (404), а не в 500.
func (h UpdateHandler) Handle(ctx context.Context, cmd UpdateTransaction) (db.Transaction, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return db.Transaction{}, ErrEmptyTitle
	}

	if cmd.AmountMinor <= 0 {
		return db.Transaction{}, ErrAmountNotPositive
	}

	if !cmd.Counted && cmd.Direction != db.TxDirectionExpense {
		return db.Transaction{}, ErrIncomeAlwaysCounted
	}

	// Категория должна существовать и быть доступна юзеру (общая или его личная).
	visible, err := h.q.CategoryVisibleToUser(ctx, db.CategoryVisibleToUserParams{
		CategoryID: ids.ToPg(cmd.CategoryID),
		UserID:     ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return db.Transaction{}, err
	}

	if !visible {
		return db.Transaction{}, ErrInvalidCategory
	}

	// Направление траты должно совпадать с направлением категории.
	catDir, err := h.q.CategoryDirection(ctx, ids.ToPg(cmd.CategoryID))
	if db.IsNotFound(err) {
		return db.Transaction{}, ErrInvalidCategory
	}

	if err != nil {
		return db.Transaction{}, err
	}

	if catDir != cmd.Direction {
		return db.Transaction{}, ErrCategoryDirectionMismatch
	}

	tx, err := h.q.UpdateTransaction(ctx, db.UpdateTransactionParams{
		ID:          ids.ToPg(cmd.ID),
		UserID:      ids.ToPg(cmd.UserID),
		CategoryID:  ids.ToPg(cmd.CategoryID),
		AmountMinor: cmd.AmountMinor,
		Title:       title,
		SpentAt:     db.Time(cmd.SpentAt),
		Direction:   cmd.Direction,
		Counted:     cmd.Counted,
	})
	if db.IsNotFound(err) {
		return db.Transaction{}, ErrNotFound
	}

	if err != nil {
		return db.Transaction{}, err
	}

	return tx, nil
}
