package balance

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/balance", c.getBalance)
	protected.PUT("/balance", c.setBalance)
}
