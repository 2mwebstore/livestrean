package repositories

import (
	"context"

	"gorm.io/gorm"

	"livestream/internal/models"
)

type RoleRepository interface {
	ListWithPermissions(ctx context.Context) ([]models.Role, error)
}

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) RoleRepository { return &roleRepository{db: db} }

func (r *roleRepository) ListWithPermissions(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).Preload("Permissions").Order("id").Find(&roles).Error
	return roles, err
}
