package initial

import (
	"pickup/internal/global"
	"pickup/pkg/models"
	"sync"
	"time"
)

func StoreUserToken(userId string, token string) {
	now := time.Now()
	global.UserTokenMap.Store(userId, models.TokenInfo{
		Token:            token,
		LastActivityTime: now,
		ExpirationTime:   now.Add(24 * time.Hour),
	})
}

func CleanupExpiredTokens(tokenMap *sync.Map) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		tokenMap.Range(func(key, value interface{}) bool {
			tokenInfo, ok := value.(models.TokenInfo)
			if !ok {
				return true
			}
			if now.After(tokenInfo.ExpirationTime) {
				tokenMap.Delete(key)
			}
			return true
		})
	}
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
	now := time.Now()
	if now.After(tokenInfo.ExpirationTime) {
		global.UserTokenMap.Delete(userId)
		return false
	}
	if tokenInfo.Token == token {

		tokenInfo.LastActivityTime = now
		global.UserTokenMap.Store(userId, tokenInfo)
		return true
	}
	return false
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
	now := time.Now()
	if now.After(tokenInfo.ExpirationTime) {
		global.UserTokenMap.Delete(userId)
		return false
	}
	tokenInfo.ExpirationTime = now.Add(duration)
	tokenInfo.LastActivityTime = now
	global.UserTokenMap.Store(userId, tokenInfo)
	return true
}
