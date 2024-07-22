package routers

import (
	"github.com/gin-gonic/gin"
	"pickup/internal/api"
)

func InitConfigRouter(router *gin.RouterGroup) {
	{
		Router := router.Group("/config")
		Router.GET("/js", api.HandleGetConfig)
	}
}
