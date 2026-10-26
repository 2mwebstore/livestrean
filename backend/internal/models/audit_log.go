package models

import "time"

type AuditLog struct {
	ID         uint64
	UserID     *uint64
	Action     string
	Resource   string
	ResourceID string
	IP         string
	UserAgent  string
	Success    bool
	CreatedAt  time.Time
}
