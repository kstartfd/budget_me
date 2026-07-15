package queries

import (
	// Стандартная библиотека.
	"context"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// TransactionRecent — запрос: последние Limit трат юзера. Limit УЖЕ разобран и
// зажат в разумные рамки на HTTP-краю (контроллер), сюда приходит готовое число —
// поэтому у запроса нет ни дефолта, ни валидации лимита. Диапазон дат тут не при
// чём: «свежее» сортируется по spent_at DESC прямо в SQL.
type TransactionRecent struct {
	UserID uuid.UUID
	Limit  int32
}

// RecentHandler — обработчик запроса. Часы не нужны (нет резолва диапазона), поэтому
// зависимость ровно одна — sqlc-запросы. PHP-аналогия — хендлер с единственной
// инъекцией репозитория в конструкторе.
type RecentHandler struct {
	q *db.Queries
}

func NewRecentHandler(q *db.Queries) RecentHandler {
	return RecentHandler{q: q}
}

// Handle — тонкий проброс в sqlc: конвертит userID в pgtype.UUID и зовёт
// сгенеренный ListRecentTransactions. Доменной логики тут нет — вся «умность»
// (дефолт/зажим лимита) осталась на HTTP-краю.
func (h RecentHandler) Handle(ctx context.Context, query TransactionRecent) ([]db.Transaction, error) {
	return h.q.ListRecentTransactions(ctx, db.ListRecentTransactionsParams{
		UserID: ids.ToPg(query.UserID),
		Lim:    query.Limit,
	})
}
