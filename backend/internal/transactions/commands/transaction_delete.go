package commands

import (
	// Стандартная библиотека.
	"context"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// Нет своей строки для удаления → её либо не было, либо она не твоя (ownership по
// user_id). Для клиента это 404; httpx сам замапит Kind NotFound на HTTP 404. В PHP
// ты бы кинул NotFoundException, а middleware перевёл бы в 404; тут — вернул значение.
var ErrNotFound = apperr.NewNotFound("transaction not found")

// DeleteTransaction — команда: КОГО удаляем. Чистый «пакет полей» без поведения.
// Роль («это команда») несёт пакет commands — поэтому суффикс Command не нужен.
type DeleteTransaction struct {
	UserID uuid.UUID
	ID     uuid.UUID
}

// DeleteHandler — обработчик команды. Часы ему не нужны (удаление их не трогает),
// поэтому держит только sqlc-запросы.
type DeleteHandler struct {
	q *db.Queries
}

func NewDeleteHandler(q *db.Queries) DeleteHandler {
	return DeleteHandler{q: q}
}

// Handle сносит ровно свою строку. n — сколько строк реально удалил DELETE
// (:execrows возвращает число затронутых строк). 0 → не нашли/не твоя → ErrNotFound.
// Отдельный SELECT для проверки владения не нужен: всё уже в WHERE (user_id = $).
func (h DeleteHandler) Handle(ctx context.Context, cmd DeleteTransaction) error {
	n, err := h.q.DeleteTransaction(ctx, db.DeleteTransactionParams{
		ID:     ids.ToPg(cmd.ID),
		UserID: ids.ToPg(cmd.UserID),
	})
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrNotFound
	}

	return nil
}
