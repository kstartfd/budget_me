package categories

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/categories", c.getCategoryList)
	protected.POST("/categories", c.createCategory)
	protected.PUT("/categories/:id", c.updateCategory)
	protected.DELETE("/categories/:id", c.deleteCategory)
}
