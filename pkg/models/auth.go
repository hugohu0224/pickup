package models

import "time"

type TokenInfo struct {
	Token            string
	LastActivityTime time.Time
	ExpirationTime   time.Time
}
