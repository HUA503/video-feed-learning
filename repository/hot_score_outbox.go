package repository

import (
	"context"
	"time"

	"example.com/video-feed/model"
	"gorm.io/gorm"
)

type HotScoreOutboxRepository struct {
	db *gorm.DB
}

func NewHotScoreOutboxRepository(
	db *gorm.DB,
) *HotScoreOutboxRepository {
	return &HotScoreOutboxRepository{db: db}
}

func (r *HotScoreOutboxRepository) Create(
	event *model.HotScoreOutbox,
) error {
	return r.db.Create(event).Error
}

// 从 Outbox 表里取出待发送事件
func (r *HotScoreOutboxRepository) ListPending(
	ctx context.Context,
	limit int,
) ([]model.HotScoreOutbox, error) {
	events := make([]model.HotScoreOutbox, 0)

	if limit <= 0 {
		return events, nil
	}

	err := r.db.
		WithContext(ctx).
		Where("published_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Find(&events). //放进 events
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// 记录某条事件已经发布完成
func (r *HotScoreOutboxRepository) MarkPublished(
	ctx context.Context,
	id uint,
) (bool, error) {
	result := r.db.
		WithContext(ctx).
		Model(&model.HotScoreOutbox{}).
		Where("id = ? AND published_at IS NULL", id).
		Update("published_at", time.Now())

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}
