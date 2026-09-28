package repository

import (
	"example.com/video-feed/model"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {

	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// 按照视频 ID，从数据库中取出属于该视频的评论，并按发布时间从早到晚排列
func (r *CommentRepository) FindByVideoID(videoID uint) ([]model.Comment, error) {
	var comments []model.Comment

	err := r.db.
		Where("video_id = ?", videoID).
		Order("created_at ASC").
		Find(&comments).
		Error

	return comments, err
}

// 查找评论
func (r *CommentRepository) FindByID(id uint) (*model.Comment, error) {
	var comment model.Comment

	if err := r.db.First(&comment, id).Error; err != nil {
		return nil, err
	}

	return &comment, nil
}

// 删除评论
func (r *CommentRepository) DeleteByID(id uint) error {
	return r.db.Delete(&model.Comment{}, id).Error
}
