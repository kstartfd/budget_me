package plans

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/plans", c.getPlanList)
	protected.GET("/plans/suggestions", c.getPlanSuggestions)
	protected.POST("/plans", c.createPlan)
	protected.PUT("/plans/:id", c.updatePlan)
	protected.POST("/plans/:id/confirm", c.confirmPlan)
	protected.DELETE("/plans/:id", c.deletePlan)
}
