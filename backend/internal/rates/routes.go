package rates

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(public, _ *echo.Group) {
	public.GET("/rates", c.getRateList)
}
