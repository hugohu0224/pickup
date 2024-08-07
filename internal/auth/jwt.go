package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"os"
	"pickup/internal/global"
	"pickup/pkg/models"
	"time"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userId string) (string, error) {
	claims := CustomClaims{
		UserID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	return token.SignedString(secretKey)
}

func ValidateJWT(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func StoreUserToken(userId string, token string) {
	jwtToken, err := GenerateJWT(userId)
	if err != nil {
		return
	}

	now := time.Now()
	global.UserTokenMap.Store(userId, models.TokenInfo{
		Token:            jwtToken,
		LastActivityTime: now,
		ExpirationTime:   now.Add(24 * time.Hour),
	})
}

func IsValidUserToken(userId string, token string) bool {
	value, ok := global.UserTokenMap.Load(userId)
	if !ok {
		return false
	}
	tokenInfo, ok := value.(models.TokenInfo)
	if !ok {
		return false
	}

	claims, err := ValidateJWT(tokenInfo.Token)
	if err != nil || claims.UserID != userId {
		global.UserTokenMap.Delete(userId)
		return false
	}

	now := time.Now()
	if now.After(tokenInfo.ExpirationTime) {
		global.UserTokenMap.Delete(userId)
		return false
	}

	tokenInfo.LastActivityTime = now
	global.UserTokenMap.Store(userId, tokenInfo)
	return true
}

func ExtendTokenExpiration(userId string, duration time.Duration) bool {
	value, ok := global.UserTokenMap.Load(userId)
	if !ok {
		return false
	}
	tokenInfo, ok := value.(models.TokenInfo)
	if !ok {
		return false
	}

	claims, err := ValidateJWT(tokenInfo.Token)
	if err != nil || claims.UserID != userId {
		global.UserTokenMap.Delete(userId)
		return false
	}

	now := time.Now()
	newExpirationTime := now.Add(duration)

	newToken, err := GenerateJWT(userId)
	if err != nil {
		return false
	}

	tokenInfo.Token = newToken
	tokenInfo.ExpirationTime = newExpirationTime
	tokenInfo.LastActivityTime = now
	global.UserTokenMap.Store(userId, tokenInfo)
	return true
}

func LoadOrGenerateJWTSecret() []byte {
	secretKeyFile := "jwt_secret.key"
	if keyBytes, err := os.ReadFile(secretKeyFile); err == nil {
		return keyBytes
	}

	// generate
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		zap.S().Panicf("cannot generate JWT secret key: %v", err)
	}
	keyString := base64.StdEncoding.EncodeToString(key)

	// store
	if err := os.WriteFile(secretKeyFile, []byte(keyString), 0600); err != nil {
		zap.S().Panicf("cannot store JWT secret key: %v", err)
	}

	return []byte(keyString)
}
