package models

import "time"

type TokenInfo struct {
	Token      string
	ExpireTime time.Time
}
