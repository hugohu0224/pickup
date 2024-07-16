package routers

import (
	"github.com/gin-gonic/gin"
	"pickup/internal/api"
)

func InitAuthRouter(router *gin.RouterGroup) {
	{
		Router := router.Group("/auth")
		Router.Static("/static", "./internal/static")
		Router.GET("/login", api.GetLoginPage)
		Router.GET("/google", api.RedirectToGoogleAuth)
		Router.GET("/google/callback", api.GoogleAuthCallback)
	}
}
