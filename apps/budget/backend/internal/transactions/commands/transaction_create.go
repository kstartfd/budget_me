package commands

import (
	// Стандартная библиотека.
	"context"
	"strings"
	"time"

	// Третьесторонние пакеты.
	"github.com/google/uuid"

	// Внутренние пакеты.
	"budget_me/backend/internal/apperr"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

// Доменные ошибки команды. apperr.NewInvalid = «клиент сам может поправить» —
// транспорт (httpx) сам замапит Kind Invalid на HTTP 400. В PHP ты бы кинул
// свой DomainException, а middleware перевёл бы его в 400; тут — возвращаешь
// значение, без throw.
var (
	ErrEmptyTitle                = apperr.NewInvalid("title must not be empty")
	ErrAmountNotPositive         = apperr.NewInvalid("amount must be positive")
	ErrInvalidCategory           = apperr.NewInvalid("category not found")
	ErrCategoryDirectionMismatch = apperr.NewInvalid("category direction does not match the transaction")
	ErrIncomeAlwaysCounted       = apperr.NewInvalid("only an expense can be left out of the calculation")
)

// TransactionCreate — команда: ЧТО создаём. Чистый «пакет полей» без поведения —
// в PHP ближе всего readonly-DTO, который отдаёшь в хендлер:
//
//	// PHP
//	final class TransactionCreate {
//	    public function __construct(
//	        public UuidInterface $userId,
//	        public UuidInterface $categoryId,
//	        public int $amountMinor,          // копейки, всегда > 0
//	        public string $title,
//	        public DateTimeImmutable $spentAt,
//	        public string $direction,         // 'income' | 'expense'
//	        public bool $counted,             // участвует ли в расчёте
//	    ) {}
//	}
//
// Роль («это команда») несёт ПАКЕТ commands — поэтому суффикс Command не нужен,
// иначе было бы заикание commands.TransactionCreateCommand.
type CreateTransaction struct {
	UserID      uuid.UUID
	CategoryID  uuid.UUID
	AmountMinor int64
	Title       string
	SpentAt     time.Time
	Direction   db.TxDirection
	// Counted — участвует ли трата в РАСЧЁТЕ (Остаток, «Свободно», перенос).
	// false — «обязательный платёж»: в статистике он есть, из денег не вычитается.
	Counted bool
}

// CreateHandler — обработчик команды. Держит ровно свои зависимости: sqlc-запросы
// и часы. PHP-аналогия — класс-хендлер с инъекцией через конструктор:
//
//	// PHP
//	final class CreateHandler {
//	    public function __construct(private Queries $q, private Clock $clock) {}
//	}
type CreateHandler struct {
	q     *db.Queries
	clock clock.Clock
}

// NewHandler-конструктора в Go нет (нет __construct) — по соглашению пишут New<Тип>.
func NewCreateHandler(q *db.Queries, clk clock.Clock) CreateHandler {
	return CreateHandler{q: q, clock: clk}
}

// Handle проверяет инварианты и вставляет строку. Приёмник «(h CreateHandler)»
// слева от имени — это как $this в PHP-методе, только пишется явно. Ошибка —
// второй результат, поток управления всегда явный (if err != nil).
func (h CreateHandler) Handle(ctx context.Context, cmd CreateTransaction) (db.Transaction, error) {
	title := strings.TrimSpace(cmd.Title) // trim() в PHP
	if title == "" {
		return db.Transaction{}, ErrEmptyTitle
	}

	// Сумма — всегда положительная величина. Знак (расход/приход) несёт Direction,
	// а не минус в amount (см. миграцию 0007).
	if cmd.AmountMinor <= 0 {
		return db.Transaction{}, ErrAmountNotPositive
	}

	// Из расчёта можно убрать только расход: приход всегда пополняет деньги.
	if !cmd.Counted && cmd.Direction != db.TxDirectionExpense {
		return db.Transaction{}, ErrIncomeAlwaysCounted
	}

	// Категория должна существовать и быть доступна юзеру (общая или его личная).
	// Один SELECT EXISTS — и наличие, и владение сразу.
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

	// Направление траты должно совпадать с направлением категории (income-трата →
	// только income-категория). db.IsNotFound — рукописный хелпер, чтобы не тащить
	// сюда драйвер pgx: команда про HTTP/драйвер не знает.
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

	// Единственная запись в БД. ids.ToPg/db.Time/ids.New — хелперы: обычные
	// Go-типы → pgtype-обёртки, которые ждёт сгенерённый sqlc-код; ids.New() —
	// новый UUID v7.
	return h.q.InsertTransaction(ctx, db.InsertTransactionParams{
		ID:          ids.ToPg(ids.New()),
		UserID:      ids.ToPg(cmd.UserID),
		CategoryID:  ids.ToPg(cmd.CategoryID),
		AmountMinor: cmd.AmountMinor,
		Title:       title,
		SpentAt:     db.Time(cmd.SpentAt),
		CreatedAt:   db.Time(h.clock.Now()),
		Direction:   cmd.Direction,
		Counted:     cmd.Counted,
	})
}
