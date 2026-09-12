package lists

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/lists", c.getListList)
	protected.POST("/lists", c.createList)
	protected.DELETE("/lists/:id", c.deleteList)
}
