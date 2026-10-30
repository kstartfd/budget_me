package learningai

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// completeBodyLimit covers the prompt cap in runes at two bytes each for
// Cyrillic text, plus the JSON around it.
const completeBodyLimit = "192K"

// The picture is sent already shrunk by the app; this leaves room for a phone's
// own file picked as it is. maxImageBytes is the picture alone, the body limit
// adds the form around it, and maxPixels is what it may unpack to: the reader
// holds the whole picture in memory.
const (
	scanBodyLimit = "7M"
	maxImageBytes = 6 << 20
	maxPixels     = 16_000_000
	badPicture    = "the picture must be a JPEG, PNG or GIF up to 6 MB"
)

var imageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true}

func (c *Controller) Routes(protected *echo.Group) {
	protected.GET("/learning/ai/settings", c.get)
	protected.PUT("/learning/ai/settings", c.save)
	protected.DELETE("/learning/ai/settings", c.remove)
	protected.POST("/learning/ai/complete", c.answer, echomw.BodyLimit(completeBodyLimit))
	protected.POST("/learning/ai/scan", c.scanPicture, echomw.BodyLimit(scanBodyLimit))
}
