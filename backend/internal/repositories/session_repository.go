package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"livestream/internal/models"
)

type SessionRepository interface {
	Create(ctx context.Context, s *models.Session) error
	FindByTokenHash(ctx context.Context, hash string) (*models.Session, error)
	Touch(ctx context.Context, id uint64, at time.Time) error
	DeleteByTokenHash(ctx context.Context, hash string) error
	DeleteByUserID(ctx context.Context, userID uint64) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type sessionRepository struct{ db *gorm.DB }

func NewSessionRepository(db *gorm.DB) SessionRepository { return &sessionRepository{db: db} }

func (r *sessionRepository) Create(ctx context.Context, s *models.Session) error {
	return translate(r.db.WithContext(ctx).Create(s).Error)
}

func (r *sessionRepository) FindByTokenHash(ctx context.Context, hash string) (*models.Session, error) {
	return first[models.Session](r.db.WithContext(ctx).Where("token_hash = ?", hash))
}

func (r *sessionRepository) Touch(ctx context.Context, id uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", id).Update("last_seen_at", at).Error
}

func (r *sessionRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", hash).Delete(&models.Session{}).Error
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.Session{}).Error
}

func (r *sessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&models.Session{})
	return res.RowsAffected, res.Error
}
