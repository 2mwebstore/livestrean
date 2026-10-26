package models

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        uint64
	ChannelID uint64
	UserID    uint64
	User      User
	Body      string
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
}
