package budgets

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/budgets", c.getBudgetList)
	protected.GET("/budgets/recommendations", c.getBudgetRecommendations)
	protected.POST("/budgets", c.saveBudget)
	protected.DELETE("/budgets/:categoryId", c.deleteBudget)
}
