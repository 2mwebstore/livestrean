// Package models maps database tables. Models are never returned from the
// API directly; controllers respond with DTOs.
package models

type Role struct {
	ID          uint64
	Slug        string
	Name        string
	Permissions []Permission `gorm:"many2many:role_permissions"`
}

type Permission struct {
	ID          uint64
	Slug        string
	Description string
}
