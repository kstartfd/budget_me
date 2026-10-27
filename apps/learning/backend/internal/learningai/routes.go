package learningai

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/ai/settings", c.get)
	protected.PUT("/learning/ai/settings", c.save)
	protected.DELETE("/learning/ai/settings", c.remove)
}
