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
	protected.POST("/learning/path/levels", c.chooseLevel, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/tests/:id", c.testState)
	protected.POST("/learning/path/tests/:id/answers", c.answer, echomw.BodyLimit("64K"))
	protected.POST("/learning/path/tests/:id/analyze", c.analyze)
	protected.GET("/learning/path/jobs/:id", c.job)
	protected.GET("/learning/path/teachers", c.listTeachers)
	protected.POST("/learning/path/teachers", c.createTeacher, echomw.BodyLimit("16K"))
	protected.DELETE("/learning/path/teachers/:id", c.deleteTeacher)
	protected.GET("/learning/path", c.current)
	protected.GET("/learning/path/history", c.history)
	protected.DELETE("/learning/path/history", c.clearHistory)
	protected.POST("/learning/path/reset", c.reset, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/rules", c.listRules)
	protected.GET("/learning/path/rules/:id", c.getRule)
	protected.GET("/learning/path/rules/:id/practice", c.practice)
	protected.POST("/learning/path/rules/:id/practice/examples", c.writeExamples)
	protected.POST("/learning/path/rules/:id/practice/exercises", c.writeExercises)
	protected.POST("/learning/path/rules/:id/practice/leave", c.leavePractice)
	protected.POST("/learning/path/rules/:id/practice/check", c.checkPractice, echomw.BodyLimit("16K"))
	protected.POST("/learning/path/rules/:id/practice/key", c.practiceKey, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/topics/:id", c.topic)
	protected.POST("/learning/path/topics/:id/exercises", c.generate)
	protected.POST("/learning/path/topics/:id/complete", c.completeTopic)
	protected.POST("/learning/path/lessons", c.startLesson)
	protected.GET("/learning/path/lessons", c.listLessons)
	protected.GET("/learning/path/lessons/active", c.activeLesson)
	protected.POST("/learning/path/lessons/:id/pause", c.pauseLesson)
	protected.POST("/learning/path/lessons/:id/resume", c.resumeLesson)
	protected.POST("/learning/path/lessons/:id/finish", c.finishLesson)
	protected.POST("/learning/path/exercises/:id/attempt", c.attempt, echomw.BodyLimit("16K"))
	protected.GET("/learning/path/exercises/:id/audio", c.exerciseAudio)
	protected.GET("/learning/path/topics/:id/words/:index/audio", c.wordAudio)
}
