package initial

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"pickup/internal/routers"
)

func InitRouters() *gin.Engine {
	r := gin.Default()
	r.Use(cors.Default())
	r.LoadHTMLGlob("./internal/templates/*")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	ApiGroup := r.Group("/v1")

	routers.InitGameRouter(ApiGroup)
	routers.InitAuthRouter(ApiGroup)

	return r
}
