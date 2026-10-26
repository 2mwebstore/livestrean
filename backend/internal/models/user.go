package models

import "time"

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type User struct {
	ID           uint64
	Name         string
	Email        string
	PasswordHash string
	RoleID       uint64
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
