package service

import (
	"context"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
)

type HotFeedService struct {
	hotVideoRepo *repository.HotVideoRepository
	videoRepo    *repository.VideoRepository
}

func NewHotFeedService(
	hotVideoRepo *repository.HotVideoRepository,
	videoRepo *repository.VideoRepository,
) *HotFeedService {
	return &HotFeedService{
		hotVideoRepo: hotVideoRepo,
		videoRepo:    videoRepo,
	}
}

// TopVideoIDs 从 Redis 取出按热度排序的视频 ID
// FindByIDs 到 MySQL 查询这些视频的完整信息
func (s *HotFeedService) TopVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, error) {
	//读取热度最高的视频 ID
	videoIDs, err := s.hotVideoRepo.TopVideoIDs(ctx, limit)
	if err != nil {
		return nil, err
	}

	if len(videoIDs) == 0 {
		return make([]model.Video, 0), nil
	}

	//去 MySQL 查询视频详情
	videos, err := s.videoRepo.FindByIDs(videoIDs)
	if err != nil {
		return nil, err
	}

	//按视频 ID 建一个查找表
	videoByID := make(map[uint]model.Video, len(videos))
	for _, video := range videos {
		videoByID[video.ID] = video
	}

	//按 Redis 给出的热门顺序重新排列视频
	orderedVideos := make([]model.Video, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		if video, exists := videoByID[videoID]; exists {
			orderedVideos = append(orderedVideos, video)
		}
	}

	return orderedVideos, nil
}
