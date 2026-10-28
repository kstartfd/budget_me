package learningai

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// completeBodyLimit covers the prompt cap in runes at two bytes each for
// Cyrillic text, plus the JSON around it.
const completeBodyLimit = "192K"

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/ai/settings", c.get)
	protected.PUT("/learning/ai/settings", c.save)
	protected.DELETE("/learning/ai/settings", c.remove)
	protected.POST("/learning/ai/complete", c.answer, echomw.BodyLimit(completeBodyLimit))
}
