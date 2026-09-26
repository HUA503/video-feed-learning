package repository

import (
	"example.com/video-feed/model"
	"gorm.io/gorm"
)

// 读写视频数据
type VideoRepository struct {
	db *gorm.DB
}

// 保存数据库连接
func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// 通过 GORM 写入视频记录并把错误交回上层
func (r *VideoRepository) Create(video *model.Video) error {
	return r.db.Create(video).Error
}
