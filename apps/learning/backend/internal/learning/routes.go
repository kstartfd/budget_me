package learning

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(group *echo.Group) {
	group.GET("/topics", c.listTopics)
	group.POST("/topics", c.saveTopic)
	group.PUT("/topics/:id", c.saveTopic)
	group.DELETE("/topics/:id", c.deleteTopic)
	group.GET("/entries", c.listEntries)
	group.GET("/entries/:id", c.getEntry)
	group.GET("/entries/:id/audio", c.getEntryAudio)
	group.POST("/entries", c.saveEntry)
	group.PUT("/entries/:id", c.saveEntry)
	group.DELETE("/entries/:id", c.deleteEntry)
	group.POST("/captures", c.enqueue)
	group.GET("/captures/:id", c.getCapture)
}
