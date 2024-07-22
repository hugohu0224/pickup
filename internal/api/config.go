package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"pickup/internal/global"
)

func GetConfigAsJSON() ([]byte, error) {
	config := global.Dv.AllSettings()
	return json.Marshal(config)
}

func HandleGetConfig(c *gin.Context) {
	jsonData, err := GetConfigAsJSON()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate config JSON"})
		return
	}
	c.Header("Content-Type", "application/json")
	c.String(200, string(jsonData))
	zap.S().Infof("json: %s", string(jsonData))

}
