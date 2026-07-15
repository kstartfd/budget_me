// Package transactions — HTTP-фича «траты» в целевой форме B: тонкий Controller,
// а вся работа — в под-пакетах commands/ (запись) и queries/ (чтение). Controller
// только разбирает запрос (shape-валидация HTTP-края), собирает вход команды/запроса,
// зовёт Handle и маппит результат в DTO. Своей бизнес-логики он не держит.
//
// PHP-аналогия: тонкий контроллер, который парсит Request, отдаёт CommandHandler /
// QueryHandler и заворачивает ответ в Resource. Инъекция зависимостей — через
// конструктор (NewController), как в PHP через __construct.
package transactions

import (
	// Стандартная библиотека.
	"strconv"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	// Внутренние пакеты.
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/httpx"
	"budget_me/backend/internal/transactions/commands"
	"budget_me/backend/internal/transactions/queries"
)

// Рамки для ?limit эндпоинта /recent. default — если параметра нет или он кривой;
// min/max — в них зажимаем переданное число (защита от 0/отрицательных и «выкачай всё»).
const (
	recentDefaultLimit int32 = 10
	recentMinLimit     int32 = 1
	recentMaxLimit     int32 = 100
)

// Controller держит по одному обработчику на операцию. Значения (не указатели):
// хендлеры — лёгкие структуры «запросы + часы», копировать их дёшево.
type Controller struct {
	create     commands.CreateHandler
	remove     commands.DeleteHandler
	update     commands.UpdateHandler
	list       queries.ListHandler
	summary    queries.SummaryHandler
	spent      queries.SpentHandler
	recent     queries.RecentHandler
	byCategory queries.ByCategoryHandler
	trend      queries.TrendHandler
}

// NewController собирает Controller: раздаёт общие зависимости (sqlc-запросы + часы)
// каждому обработчику. RecentHandler часы не нужны — ему передаём только запросы.
func NewController(q *db.Queries, clk clock.Clock) *Controller {
	return &Controller{
		create:     commands.NewCreateHandler(q, clk),
		remove:     commands.NewDeleteHandler(q),
		update:     commands.NewUpdateHandler(q),
		list:       queries.NewListHandler(q, clk),
		summary:    queries.NewSummaryHandler(q, clk),
		spent:      queries.NewSpentHandler(q, clk),
		recent:     queries.NewRecentHandler(q),
		byCategory: queries.NewByCategoryHandler(q, clk),
		trend:      queries.NewTrendHandler(q, clk),
	}
}

// getTransactionList — GET /transactions?from=&to=. userID кладёт middleware
// RequireAuth (роут protected — юзер всегда есть). from/to опциональны: пустые —
// не ошибка, дефолт подставит resolveRange в запросе.
func (c *Controller) getTransactionList(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	txs, err := c.list.Handle(ctx.Request().Context(), queries.TransactionList{
		UserID: userID,
		From:   ctx.QueryParam("from"),
		To:     ctx.QueryParam("to"),
	})
	if err != nil {
		return err
	}

	out := make([]transactionResponse, len(txs))
	for i, t := range txs {
		out[i] = toTransactionResponse(t)
	}

	return httpx.OK(ctx, out)
}

// createTransaction — POST /transactions. Здесь только shape-валидация HTTP-края:
// categoryId → uuid, spentAt (RFC3339 UTC) → момент, direction → один из двух enum. Бизнес-правила
// (title непустой, amount>0, категория видима и того же направления) проверит команда.
func (c *Controller) createTransaction(ctx echo.Context) error {
	var req transactionRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	// categoryId — строка на проводе → uuid. Кривой uuid не доходит до команды.
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return apperr.NewInvalid("categoryId must be a valid uuid")
	}

	// Момент траты приходит в UTC (RFC3339) — никакой сборки из кусков и никакого
	// угадывания пояса на бэке. PHP-аналогия: new DateTimeImmutable($req->spentAt).
	spentAt, err := time.Parse(time.RFC3339, req.SpentAt)
	if err != nil {
		return apperr.NewInvalid("bad spentAt (want RFC3339, e.g. 2026-07-13T13:30:00Z)")
	}

	// direction опционален (фронт слал только расходы): пусто → expense; иначе принимаем
	// лишь два enum-значения. db.ParseDirection — общий маппер строки в TxDirection.
	direction := db.TxDirectionExpense

	if req.Direction != "" {
		d, ok := db.ParseDirection(req.Direction)
		if !ok {
			return apperr.NewInvalid(`direction must be "income" or "expense"`)
		}

		direction = d
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	tx, err := c.create.Handle(ctx.Request().Context(), commands.CreateTransaction{
		UserID:      userID,
		CategoryID:  categoryID,
		AmountMinor: req.AmountMinor,
		Title:       req.Title,
		SpentAt:     spentAt,
		Direction:   direction,
	})
	if err != nil {
		return err
	}

	return httpx.Created(ctx, toTransactionResponse(tx))
}

// deleteTransaction — DELETE /transactions/:id. Shape-валидация края: :id → uuid.
// Кривой uuid не доходит до команды. Владение (строка твоя) проверяет команда через
// WHERE user_id = $ — чужую/несуществующую трату вернёт как 404 (ErrNotFound).
func (c *Controller) deleteTransaction(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid transaction id")
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	if err := c.remove.Handle(ctx.Request().Context(), commands.DeleteTransaction{
		UserID: userID,
		ID:     id,
	}); err != nil {
		return err
	}

	return httpx.OK(ctx, echo.Map{"id": id.String()})
}

// updateTransaction — PUT /transactions/:id. Тот же разбор края, что и create
// (categoryId→uuid, spentAt→момент, direction→enum), плюс :id из пути. Все
// бизнес-правила и «строка твоя?» проверяет команда (чужую/несуществующую → 404).
func (c *Controller) updateTransaction(ctx echo.Context) error {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return apperr.NewInvalid("invalid transaction id")
	}

	var req transactionRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return apperr.NewInvalid("categoryId must be a valid uuid")
	}

	spentAt, err := time.Parse(time.RFC3339, req.SpentAt)
	if err != nil {
		return apperr.NewInvalid("bad spentAt (want RFC3339, e.g. 2026-07-13T13:30:00Z)")
	}

	direction := db.TxDirectionExpense

	if req.Direction != "" {
		d, ok := db.ParseDirection(req.Direction)
		if !ok {
			return apperr.NewInvalid(`direction must be "income" or "expense"`)
		}

		direction = d
	}

	userID := httpx.UserIDFromContext(ctx.Request().Context())

	tx, err := c.update.Handle(ctx.Request().Context(), commands.UpdateTransaction{
		UserID:      userID,
		ID:          id,
		CategoryID:  categoryID,
		AmountMinor: req.AmountMinor,
		Title:       req.Title,
		SpentAt:     spentAt,
		Direction:   direction,
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toTransactionResponse(tx))
}

// getTransactionSummary — GET /transactions/summary?from=&to=: расход, приход и разбивка
// расхода по категориям одним запросом. NULL-бакет уже отфильтрован в queries.
func (c *Controller) getTransactionSummary(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	res, err := c.summary.Handle(ctx.Request().Context(), queries.TransactionSummary{
		UserID: userID,
		From:   ctx.QueryParam("from"),
		To:     ctx.QueryParam("to"),
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, toSummaryResponse(res))
}

// getTransactionSpent — GET /transactions/spent?from=&to=: сумма расхода за период.
func (c *Controller) getTransactionSpent(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	n, err := c.spent.Handle(ctx.Request().Context(), queries.TransactionSpent{
		UserID: userID,
		From:   ctx.QueryParam("from"),
		To:     ctx.QueryParam("to"),
	})
	if err != nil {
		return err
	}

	return httpx.OK(ctx, spentResponse{SpentMinor: n})
}

// getRecentTransactions — GET /transactions/recent?limit=N: последние N трат. limit
// опционален: пусто/не число → дефолт 10; валидное число зажимаем в [1, 100]. Кривой
// limit — НЕ 400, просто дефолтимся (эндпоинт «покажи свежее», строгую валидацию тут
// навязывать незачем). Это shape-нормализация HTTP-края — запрос получает готовое число.
func (c *Controller) getRecentTransactions(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	limit := recentDefaultLimit

	if raw := ctx.QueryParam("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = int32(n)
			if limit < recentMinLimit {
				limit = recentMinLimit
			}

			if limit > recentMaxLimit {
				limit = recentMaxLimit
			}
		}
	}

	txs, err := c.recent.Handle(ctx.Request().Context(), queries.TransactionRecent{
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return err
	}

	out := make([]transactionResponse, len(txs))
	for i, t := range txs {
		out[i] = toTransactionResponse(t)
	}

	return httpx.OK(ctx, out)
}

// getTransactionsByCategory — GET /transactions/by-category?from=&to=: суммы трат по
// категориям за период. Строки уже отсортированы (spent_minor DESC) и очищены от
// NULL-бакета в queries, поэтому здесь только маппинг в DTO.
func (c *Controller) getTransactionsByCategory(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	rows, err := c.byCategory.Handle(ctx.Request().Context(), queries.TransactionByCategory{
		UserID: userID,
		From:   ctx.QueryParam("from"),
		To:     ctx.QueryParam("to"),
	})
	if err != nil {
		return err
	}

	out := make([]categorySpendResponse, len(rows))
	for i, row := range rows {
		out[i] = toCategorySpendResponse(row)
	}

	return httpx.OK(ctx, out)
}

// getTransactionTrend — GET /transactions/trend?months=6&tz=Europe/Minsk: расход по
// последним N расчётным периодам, от старого к новому. tz присылает клиент (бэкенд
// поясов не хранит); пусто/мусор → UTC. months опционален и НЕ обязан быть
// числом-в-диапазоне: кривое значение просто отдаёт дефолт (0 → 6 в хендлере),
// а слишком большое там же подрежется до потолка. Ошибку 400 городить не за что.
func (c *Controller) getTransactionTrend(ctx echo.Context) error {
	userID := httpx.UserIDFromContext(ctx.Request().Context())

	months, _ := strconv.Atoi(ctx.QueryParam("months"))

	points, err := c.trend.Handle(ctx.Request().Context(), queries.TransactionTrend{
		UserID:   userID,
		Months:   months,
		Timezone: ctx.QueryParam("tz"),
	})
	if err != nil {
		return err
	}

	out := make([]trendPointResponse, len(points))
	for i, p := range points {
		out[i] = toTrendPointResponse(p)
	}

	return httpx.OK(ctx, out)
}
