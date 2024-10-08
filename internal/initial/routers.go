package initial

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"pickup/internal/global"
	"pickup/internal/routers"
)

func InitRouters() *gin.Engine {
	r := gin.Default()

	if global.Dv.GetBool("ALLOW_CORS") {
		r.Use(cors.Default())
	}
	r.LoadHTMLGlob("./internal/templates/*")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// default page
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/v1/auth/login")
	})

	ApiGroup := r.Group("/v1")

	routers.InitGameRouter(ApiGroup)
	routers.InitAuthRouter(ApiGroup)
	routers.InitConfigRouter(ApiGroup)
	routers.InitUserRouter(ApiGroup)

	return r
}
