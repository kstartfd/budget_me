package learningchat

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/entries/:id/chat", c.get)
	protected.POST("/learning/entries/:id/chat", c.send, echomw.BodyLimit("16K"))
	protected.DELETE("/learning/entries/:id/chat", c.clear)

	protected.GET("/learning/topics/:id/chat", c.getTopic)
	protected.POST("/learning/topics/:id/chat", c.sendTopic, echomw.BodyLimit("16K"))
	protected.POST("/learning/topics/:id/chat/archive", c.archive)
	protected.GET("/learning/topics/:id/chat/history", c.history)
	protected.GET("/learning/path/topics/:id/chat", c.getPathTopic)
	protected.POST("/learning/path/topics/:id/chat", c.sendPathTopic, echomw.BodyLimit("16K"))
	protected.DELETE("/learning/path/topics/:id/chat", c.clearPathTopic)

	protected.GET("/learning/path/rules/:id/examples/:example/chat", c.getExample)
	protected.POST("/learning/path/rules/:id/examples/:example/chat", c.sendExample, echomw.BodyLimit("16K"))

	protected.GET("/learning/chat-dialogs/:id", c.dialog)
	protected.DELETE("/learning/chat-dialogs/:id", c.deleteDialog)
}
