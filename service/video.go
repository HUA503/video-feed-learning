package service

import (
	"example.com/video-feed/model"
	"example.com/video-feed/repository"
)

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
