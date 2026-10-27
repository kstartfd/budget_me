package learningchat

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/entries/:id/chat", c.get)
	protected.POST("/learning/entries/:id/chat", c.send, echomw.BodyLimit("16K"))
	protected.DELETE("/learning/entries/:id/chat", c.clear)
}
