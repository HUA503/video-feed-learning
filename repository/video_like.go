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
