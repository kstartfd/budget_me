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

// AgentRoutes are the calls that spend the learner's own AI key. They sit on the
// account's session group, which a topic-scoped extension token does not reach.
func (c *Controller) AgentRoutes(protected *echo.Group) {
	protected.POST("/learning/entries/:id/fill", c.fillEntry)
}
