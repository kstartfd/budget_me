package items

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/items", c.getItemList)
	protected.POST("/items", c.createItem)
	protected.PUT("/items/:id", c.updateItem)
	protected.DELETE("/items/:id", c.deleteItem)
}
