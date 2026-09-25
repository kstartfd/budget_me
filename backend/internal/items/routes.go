package items

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func (c *Controller) Routes(_, protected *echo.Group) {
	protected.GET("/items", c.getItemList)
	protected.POST("/items", c.createItem)
	protected.PUT("/items/:id", c.updateItem)
	protected.DELETE("/items/:id", c.deleteItem)

	// BodyLimit above maxPhotoUpload rejects an oversized request before it is
	// even buffered into multipart parts.
	protected.POST("/items/:id/photo", c.uploadItemPhoto, echomw.BodyLimit("21M"))
	protected.DELETE("/items/:id/photo", c.deleteItemPhoto)
}
