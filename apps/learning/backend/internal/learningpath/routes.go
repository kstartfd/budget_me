package learningpath

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// Routes are session-only, like the chat and the agent settings: a path spends
// the learner's own API key, which a topic-scoped extension token must never
// be able to do.
func (c *Controller) Routes(protected *echo.Group) {
	protected.POST("/learning/path/tests", c.startTest, echomw.BodyLimit("16K"))
	protected.POST("/learning/path/final-tests", c.startFinal, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/tests/:id", c.testState)
	protected.POST("/learning/path/tests/:id/answers", c.answer, echomw.BodyLimit("64K"))
	protected.POST("/learning/path/tests/:id/analyze", c.analyze)
	protected.GET("/learning/path", c.current)
	protected.GET("/learning/path/history", c.history)
	protected.POST("/learning/path/reset", c.reset, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/topics/:id", c.topic)
	protected.POST("/learning/path/topics/:id/exercises", c.generate)
	protected.POST("/learning/path/topics/:id/complete", c.completeTopic)
	protected.POST("/learning/path/topics/:id/lessons", c.startLesson)
	protected.GET("/learning/path/topics/:id/lessons", c.listLessons)
	protected.GET("/learning/path/lessons/active", c.activeLesson)
	protected.POST("/learning/path/lessons/:id/pause", c.pauseLesson)
	protected.POST("/learning/path/lessons/:id/resume", c.resumeLesson)
	protected.POST("/learning/path/lessons/:id/finish", c.finishLesson)
	protected.POST("/learning/path/exercises/:id/attempt", c.attempt, echomw.BodyLimit("16K"))
}
