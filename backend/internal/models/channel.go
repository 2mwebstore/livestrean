package models

import "time"

type Channel struct {
	ID            uint64
	OwnerID       uint64
	Owner         User `gorm:"foreignKey:OwnerID"`
	Slug          string
	Title         string
	Description   string
	StreamKeyHash string
	Status        string
	LastLiveAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
