package routers

import (
	"github.com/gin-gonic/gin"
	"pickup/internal/api"
)

func InitGameRouter(router *gin.RouterGroup) {
	{
		Router := router.Group("/game")
		//Router.Static("/static", "./internal/static")
		Router.GET("/ws-url", api.GetWebSocketURL)
		Router.GET("/ws", api.WebsocketEndpoint)
	}
}
