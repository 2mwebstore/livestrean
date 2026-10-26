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
	// Next scheduled stream; viewers see a countdown to it.
	ScheduledAt *time.Time
	// Changes on every thumbnail upload; nil when there is no thumbnail.
	ThumbnailVersion *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ChannelThumbnail struct {
	ChannelID   uint64 `gorm:"primaryKey"`
	ContentType string
	Data        []byte
	UpdatedAt   time.Time
}
