package dto

import "time"

type AdminUserResponse struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin streamer viewer"`
}

type RoleResponse struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}
