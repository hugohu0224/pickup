package initial

import (
	"go.uber.org/zap"
	"os"
	"pickup/internal/auth"
)

func InitJWTSecretKey() {
	jwtSecretKey, err := auth.LoadOrGenerateJWTSecret()
	if err != nil {
		zap.L().Error("Error in LoadOrGenerateJWTSecret", zap.Error(err))
	}
	err = os.Setenv("JWT_SECRET_KEY", string(jwtSecretKey))
	if err != nil {
		zap.S().Errorf("failed setting JWT Secret Key")
		return
	}
}
