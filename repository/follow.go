package repository

import (
	"example.com/video-feed/model"
	"gorm.io/gorm"
)

type FollowRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Create(follow *model.Follow) error {
	return r.db.Create(follow).Error
}

// 检查被关注者是否真实存在
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User

	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// 检查两个人之间是否已经存在关注关系
func (r *FollowRepository) Exists(followerID, followeeID uint) (bool, error) {
	var count int64

	err := r.db.
		Model(&model.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&count).
		Error

	return count > 0, err
}

// 取消关注
func (r *FollowRepository) Delete(followerID, followeeID uint) error {
	return r.db.
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&model.Follow{}).
		Error
}
