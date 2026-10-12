package auth

import "github.com/labstack/echo/v4"

func (c *Controller) Routes(public, protected *echo.Group) {
	public.POST("/auth/telegram", c.authTelegram)
	public.POST("/auth/login", c.loginUser)
	protected.GET("/me", c.getMe)
	protected.POST("/auth/link/email", c.linkEmailIdentity)
}
