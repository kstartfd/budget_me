package tasks

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/tasks", c.getDayTasks)
	protected.POST("/tasks", c.createTask)
	protected.PUT("/tasks/:id", c.updateTask)
	protected.PUT("/tasks/:id/done", c.setTaskDone)
	protected.DELETE("/tasks/:id", c.deleteTask)
}
