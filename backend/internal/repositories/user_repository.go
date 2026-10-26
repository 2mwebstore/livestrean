package repositories

import (
	"context"

	"gorm.io/gorm"

	"livestream/internal/models"
)

type UserFilter struct {
	Query string
	Page  int
	Limit int
}

type UserRepository interface {
	FindByID(ctx context.Context, id uint64) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	UpdateStatus(ctx context.Context, id uint64, status string) error
	UpdateRole(ctx context.Context, id uint64, roleID uint64) error
	List(ctx context.Context, f UserFilter) ([]models.User, int64, error)
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepository{db: db} }

func (r *userRepository) FindByID(ctx context.Context, id uint64) (*models.User, error) {
	return first[models.User](r.db.WithContext(ctx).Where("id = ?", id))
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return first[models.User](r.db.WithContext(ctx).Where("email = ?", email))
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return translate(r.db.WithContext(ctx).Create(user).Error)
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return affected(r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("status", status))
}

func (r *userRepository) UpdateRole(ctx context.Context, id uint64, roleID uint64) error {
	return affected(r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("role_id", roleID))
}

func (r *userRepository) List(ctx context.Context, f UserFilter) ([]models.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.User{})
	if f.Query != "" {
		like := containsPattern(f.Query)
		q = q.Where("name LIKE ? OR email LIKE ?", like, like)
	}
	q = q.Session(&gorm.Session{}) // reused for Count and Find

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []models.User
	err := q.Select("id", "name", "email", "role_id", "status", "created_at", "updated_at").
		Order("id DESC").Offset(offset(f.Page, f.Limit)).Limit(f.Limit).
		Find(&users).Error
	return users, total, err
}
