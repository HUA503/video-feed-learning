package service

import (
	"errors"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var ErrVideoNotFound = errors.New("视频不存在")

// 处理视频业务
type VideoService struct {
	videoRepo *repository.VideoRepository
}

func NewVideoService(videoRepo *repository.VideoRepository) *VideoService {
	return &VideoService{videoRepo: videoRepo}
}

func (s *VideoService) Create(
	userID uint,
	title string,
	description string,
	videoURL string,
	coverURL string,
) (*model.Video, error) {
	video := &model.Video{
		UserID:      userID,
		Title:       title,
		Description: description,
		VideoURL:    videoURL,
		CoverURL:    coverURL,
	}

	if err := s.videoRepo.Create(video); err != nil {
		return nil, err
	}

	return video, nil
}

// /根据视频 ID 获取视频详情，并把“视频不存在”转换成项目自己的业务错误
func (s *VideoService) GetByID(id uint) (*model.Video, error) {
	video, err := s.videoRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	return video, nil
}
