package service

import (
	"context"
	"log"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"golang.org/x/sync/singleflight"
)

type HotFeedService struct {
	hotVideoRepo *repository.HotVideoRepository
	videoRepo    *repository.VideoRepository
	hotFeedGroup singleflight.Group //当前有哪些查询正在执行，哪些请求可以等待并共享结果
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

// TopWindowVideoIDs 从 Redis 窗口榜取出按热度排序的视频 ID
// FindByIDs 到 MySQL 查询这些视频的完整信息
func (s *HotFeedService) loadTopVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, error) {
	//读取热度最高的视频 ID
	videoIDs, err := s.hotVideoRepo.TopWindowVideoIDs(ctx, limit)
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

// 优先读取列表缓存，未命中时查询并回填缓存
func (s *HotFeedService) TopVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, error) {
	cachedVideos, hit, err := s.hotVideoRepo.GetCachedWindowVideos(
		ctx,
		limit,
	)
	if err != nil {
		log.Printf("读取热门列表缓存失败，继续查询榜单: %v", err)
	} else if hit {
		return cachedVideos, nil
	}

	// 没有可用缓存，执行原来的查询逻辑
	videos, err := s.loadTopVideos(ctx, limit)
	if err != nil {
		return nil, err
	}

	// 查询成功后，保存一份 5 秒缓存
	if err := s.hotVideoRepo.SetCachedWindowVideos(
		ctx,
		limit,
		videos,
	); err != nil {
		log.Printf("保存热门列表缓存失败: %v", err)
	}

	return videos, nil
}
