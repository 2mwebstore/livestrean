package repositories

import (
	"context"

	"gorm.io/gorm"

	"livestream/internal/models"
)

type CommentRepository interface {
	Create(ctx context.Context, c *models.Comment) error
	FindByID(ctx context.Context, id uint64) (*models.Comment, error)
	// ListByChannel returns up to limit comments older than beforeID
	// (0 = newest), newest first.
	ListByChannel(ctx context.Context, channelID, beforeID uint64, limit int) ([]models.Comment, error)
	Delete(ctx context.Context, id uint64) error
}

type commentRepository struct{ db *gorm.DB }

func NewCommentRepository(db *gorm.DB) CommentRepository { return &commentRepository{db: db} }

func (r *commentRepository) Create(ctx context.Context, c *models.Comment) error {
	return translate(r.db.WithContext(ctx).Omit("User").Create(c).Error)
}

func (r *commentRepository) FindByID(ctx context.Context, id uint64) (*models.Comment, error) {
	return first[models.Comment](r.db.WithContext(ctx).Where("id = ?", id))
}

func (r *commentRepository) ListByChannel(ctx context.Context, channelID, beforeID uint64, limit int) ([]models.Comment, error) {
	q := r.db.WithContext(ctx).
		Preload("User", func(q *gorm.DB) *gorm.DB { return q.Select("id", "name") }).
		Where("channel_id = ?", channelID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var list []models.Comment
	err := q.Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

// Delete soft-deletes, keeping the row for moderation history.
func (r *commentRepository) Delete(ctx context.Context, id uint64) error {
	return affected(r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Comment{}))
}
