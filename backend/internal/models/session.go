package models

import "time"

type Session struct {
	ID         uint64
	UserID     uint64
	TokenHash  string
	IP         string
	UserAgent  string
	ExpiresAt  time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
}
