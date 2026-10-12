package settings

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/settings", c.getSettings)
	protected.PUT("/settings", c.updateSettings)
}
