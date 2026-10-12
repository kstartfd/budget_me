package insights

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/insights", c.getInsights)
}
