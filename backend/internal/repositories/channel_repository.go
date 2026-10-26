package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"livestream/internal/models"
)

type ChannelFilter struct {
	Query string
	// PublicOnly limits to active channels whose owner is active.
	PublicOnly bool
	// When FilterSlugs is set, only channels in Slugs are returned.
	FilterSlugs bool
	Slugs       []string
	Page        int
	Limit       int
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *models.Channel) error
	FindByID(ctx context.Context, id uint64) (*models.Channel, error)
	FindBySlug(ctx context.Context, slug string) (*models.Channel, error)
	ListByOwner(ctx context.Context, ownerID uint64) ([]models.Channel, error)
	CountByOwner(ctx context.Context, ownerID uint64) (int64, error)
	List(ctx context.Context, f ChannelFilter) ([]models.Channel, int64, error)
	UpdateDetails(ctx context.Context, id uint64, title, description string) error
	UpdateStreamKeyHash(ctx context.Context, id uint64, hash string) error
	UpdateStatus(ctx context.Context, id uint64, status string) error
	TouchLastLive(ctx context.Context, slug string, at time.Time) error
	Delete(ctx context.Context, id uint64) error
}

type channelRepository struct{ db *gorm.DB }

func NewChannelRepository(db *gorm.DB) ChannelRepository { return &channelRepository{db: db} }

// withOwner loads the owner without sensitive columns.
func withOwner(db *gorm.DB) *gorm.DB {
	return db.Preload("Owner", func(q *gorm.DB) *gorm.DB {
		return q.Select("id", "name", "email", "status")
	})
}

func (r *channelRepository) Create(ctx context.Context, ch *models.Channel) error {
	return translate(r.db.WithContext(ctx).Omit("Owner").Create(ch).Error)
}

func (r *channelRepository) FindByID(ctx context.Context, id uint64) (*models.Channel, error) {
	return first[models.Channel](withOwner(r.db.WithContext(ctx)).Where("id = ?", id))
}

func (r *channelRepository) FindBySlug(ctx context.Context, slug string) (*models.Channel, error) {
	return first[models.Channel](withOwner(r.db.WithContext(ctx)).Where("slug = ?", slug))
}

func (r *channelRepository) ListByOwner(ctx context.Context, ownerID uint64) ([]models.Channel, error) {
	var list []models.Channel
	err := withOwner(r.db.WithContext(ctx)).Where("owner_id = ?", ownerID).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *channelRepository) CountByOwner(ctx context.Context, ownerID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Channel{}).Where("owner_id = ?", ownerID).Count(&n).Error
	return n, err
}

func (r *channelRepository) List(ctx context.Context, f ChannelFilter) ([]models.Channel, int64, error) {
	if f.FilterSlugs && len(f.Slugs) == 0 {
		return nil, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&models.Channel{})
	if f.PublicOnly {
		q = q.Joins("JOIN users ON users.id = channels.owner_id AND users.status = ?", models.StatusActive).
			Where("channels.status = ?", models.StatusActive)
	}
	if f.FilterSlugs {
		q = q.Where("channels.slug IN ?", f.Slugs)
	}
	if f.Query != "" {
		like := containsPattern(f.Query)
		q = q.Where("channels.slug LIKE ? OR channels.title LIKE ?", like, like)
	}
	q = q.Session(&gorm.Session{}) // reused for Count and Find

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Channel
	err := withOwner(q).
		Order("channels.last_live_at IS NULL, channels.last_live_at DESC, channels.id DESC").
		Offset(offset(f.Page, f.Limit)).Limit(f.Limit).
		Find(&list).Error
	return list, total, err
}

func (r *channelRepository) UpdateDetails(ctx context.Context, id uint64, title, description string) error {
	return affected(r.db.WithContext(ctx).Model(&models.Channel{}).Where("id = ?", id).
		Updates(map[string]any{"title": title, "description": description, "updated_at": time.Now().UTC()}))
}

func (r *channelRepository) UpdateStreamKeyHash(ctx context.Context, id uint64, hash string) error {
	return affected(r.db.WithContext(ctx).Model(&models.Channel{}).Where("id = ?", id).
		Updates(map[string]any{"stream_key_hash": hash, "updated_at": time.Now().UTC()}))
}

func (r *channelRepository) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return affected(r.db.WithContext(ctx).Model(&models.Channel{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()}))
}

func (r *channelRepository) TouchLastLive(ctx context.Context, slug string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Channel{}).Where("slug = ?", slug).
		UpdateColumn("last_live_at", at).Error
}

func (r *channelRepository) Delete(ctx context.Context, id uint64) error {
	return affected(r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Channel{}))
}
