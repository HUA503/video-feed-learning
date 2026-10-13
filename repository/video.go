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

// 视频详情查询
func (r *VideoRepository) FindByID(id uint) (*model.Video, error) {
	var video model.Video

	if err := r.db.First(&video, id).Error; err != nil {
		return nil, err
	}

	return &video, nil
}

// 按时间排序的视频 Feed 列表
func (r *VideoRepository) List(
	page int,
	pageSize int,
) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	if err := r.db.Model(&model.Video{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize

	if err := r.db.
		Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize). //限制每页条数
		Offset(offset).  //跳过前面已有的记录
		Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil //视频列表、视频总数、错误
}

// 只查指定用户发布的视频
func (r *VideoRepository) ListByUserIDs(
	userIDs []uint,
	page int,
	pageSize int,
) ([]model.Video, int64, error) {
	videos := make([]model.Video, 0)
	var total int64

	if len(userIDs) == 0 {
		return videos, 0, nil
	}

	query := r.db.Model(&model.Video{}).
		Where("user_id IN ?", userIDs)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize

	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// 按指定用户筛选视频，从游标位置继续向后读取
func (r *VideoRepository) ListByUserIDsAfter(
	userIDs []uint,
	cursor *model.VideoCursor,
	limit int,
) ([]model.Video, error) {
	videos := make([]model.Video, 0)

	if len(userIDs) == 0 || limit <= 0 {
		return videos, nil
	}

	query := r.db.Model(&model.Video{}).
		Where("user_id IN ?", userIDs)

	// 第一页没有游标；后续页根据上一页最后一条视频的位置筛选
	if cursor != nil {
		query = query.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&videos).
		Error

	if err != nil {
		return nil, err
	}

	return videos, nil
}
