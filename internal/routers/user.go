package routers

import (
	"github.com/gin-gonic/gin"
	"pickup/internal/api"
)

func InitUserRouter(router *gin.RouterGroup) {
	{
		Router := router.Group("/user")
		Router.Static("/static", "./internal/static")
		Router.GET("/id", api.GetUserId)
	}
}
