package repository

import (
	"example.com/video-feed/model"
	"gorm.io/gorm"
)

// 把一条点赞记录保存到 MySQL
type VideoLikeRepository struct {
	db *gorm.DB
}

func NewVideoLikeRepository(db *gorm.DB) *VideoLikeRepository {
	return &VideoLikeRepository{db: db}
}

func (r *VideoLikeRepository) Create(like *model.VideoLike) error {
	return r.db.Create(like).Error
}

// 点赞检查
func (r *VideoLikeRepository) Exists(userID, videoID uint) (bool, error) {
	var count int64

	err := r.db.
		Model(&model.VideoLike{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// 取消点赞
func (r *VideoLikeRepository) DeleteByUserAndVideo(
	userID, videoID uint,
) (bool, error) {
	result := r.db.
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Delete(&model.VideoLike{})

	if result.Error != nil {
		return false, result.Error
	}

	//实际删除了多少条记录
	return result.RowsAffected > 0, nil
}

// 在同一个事务中保存点赞记录和热度事件
func (r *VideoLikeRepository) CreateWithOutbox(
	like *model.VideoLike,
	event *model.HotScoreOutbox,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		likeRepo := NewVideoLikeRepository(tx)
		outboxRepo := NewHotScoreOutboxRepository(tx)

		// 1. 保存点赞记录
		if err := likeRepo.Create(like); err != nil {
			return err
		}

		// 2. 保存对应的热度事件
		if err := outboxRepo.Create(event); err != nil {
			return err
		}

		return nil
	})
}

// 取消点赞，同时保存减分事件
func (r *VideoLikeRepository) DeleteWithOutbox(
	userID, videoID uint,
	event *model.HotScoreOutbox,
) (bool, error) {
	var deleted bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		likeRepo := NewVideoLikeRepository(tx)

		// 1. 删除当前用户对这个视频的点赞
		removed, err := likeRepo.DeleteByUserAndVideo(
			userID,
			videoID,
		)
		if err != nil {
			return err
		}

		// 2. 没有实际删除点赞，就不保存减分事件
		if !removed {
			return nil
		}

		// 3. 在同一个事务中保存减分事件
		outboxRepo := NewHotScoreOutboxRepository(tx)
		if err := outboxRepo.Create(event); err != nil {
			return err
		}

		deleted = true
		return nil
	})

	if err != nil {
		return false, err
	}

	return deleted, nil
}
