package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"os"
	"time"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userId string, expiresMin int) (string, error) {
	claims := CustomClaims{
		UserID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresMin) * time.Minute)),
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
