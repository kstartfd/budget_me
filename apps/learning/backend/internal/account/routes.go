package account

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(protected *echo.Group) {
	protected.POST("/account/deletion", c.requestDeletion)
	protected.DELETE("/account/deletion", c.cancelDeletion)
}
