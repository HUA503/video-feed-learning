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

// 把“评论”和“对应的热度事件”一起保存到 MySQL
func (r *CommentRepository) CreateWithOutbox(
	comment *model.Comment,
	event *model.HotScoreOutbox,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		commentRepo := NewCommentRepository(tx)
		outboxRepo := NewHotScoreOutboxRepository(tx)

		if err := commentRepo.Create(comment); err != nil {
			return err
		}

		if err := outboxRepo.Create(event); err != nil {
			return err
		}

		return nil
	})
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
func (r *CommentRepository) DeleteByID(id uint) (bool, error) {
	result := r.db.Delete(&model.Comment{}, id)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// 删除评论，同时保存减分事件
func (r *CommentRepository) DeleteWithOutbox(
	commentID uint,
	event *model.HotScoreOutbox,
) (bool, error) {
	var deleted bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 在当前事务中删除评论
		result := tx.Delete(&model.Comment{}, commentID)
		if result.Error != nil {
			return result.Error
		}

		// 没有实际删除记录，就不保存减分事件
		if result.RowsAffected == 0 {
			return nil
		}

		// 在同一个事务中保存减分事件
		outboxRepo := NewHotScoreOutboxRepository(tx) //创建使用当前事务
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
