package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"pickup/internal/auth"
)

func GetUserId(c *gin.Context) {
	tokenString, err := c.Cookie("jwt")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No jwt provided"})
		return
	}

	claims, err := auth.ValidateJWT(tokenString)
	if err != nil {
		zap.S().Errorf("Error validating token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID})
}
