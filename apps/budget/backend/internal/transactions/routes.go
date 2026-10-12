package transactions

import "github.com/labstack/echo/v4"

// Routes регистрирует маршруты фичи «траты». `protected` — группа за middleware
// RequireAuth: сюда доходят ТОЛЬКО запросы с валидным токеном. Первый аргумент
// (public) не нужен — глушим `_`, транзакции всегда приватные.
func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/transactions", c.getTransactionList)                    // список за период
	protected.POST("/transactions", c.createTransaction)                    // добавить трату
	protected.PUT("/transactions/:id", c.updateTransaction)                 // изменить трату
	protected.DELETE("/transactions/:id", c.deleteTransaction)              // удалить трату
	protected.GET("/transactions/summary", c.getTransactionSummary)         // расход+приход+разбивка
	protected.GET("/transactions/totals", c.getTransactionTotals)           // расход+приход под фильтром
	protected.GET("/transactions/recent", c.getRecentTransactions)          // последние N
	protected.GET("/transactions/by-category", c.getTransactionsByCategory) // разбивка по категориям
	protected.GET("/transactions/trend", c.getTransactionTrend)             // расход по последним N периодам
}
