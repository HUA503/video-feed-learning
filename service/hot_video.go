package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"golang.org/x/sync/singleflight"
)

var ErrHotRankingUnavailable = errors.New("窗口热榜暂不可用")

type HotFeedService struct {
	hotVideoRepo *repository.HotVideoRepository
	videoRepo    *repository.VideoRepository
	hotFeedGroup singleflight.Group //当前有哪些查询正在执行，哪些请求可以等待并共享结果

	latestCacheMu sync.RWMutex //按 limit 保存不同数量的视频列表
	latestCache   map[int]latestFeedCacheEntry
}

func NewHotFeedService(
	hotVideoRepo *repository.HotVideoRepository,
	videoRepo *repository.VideoRepository,
) *HotFeedService {
	return &HotFeedService{
		hotVideoRepo: hotVideoRepo,
		videoRepo:    videoRepo,
		latestCache:  make(map[int]latestFeedCacheEntry),
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
		return nil, fmt.Errorf("%w: %w", ErrHotRankingUnavailable, err)
	}

	if len(videoIDs) == 0 {
		return make([]model.Video, 0), nil
	}

	//去 MySQL 查询视频详情
	videos, err := s.videoRepo.FindByIDs(ctx, videoIDs)
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

// 优先读取缓存，未命中时合并相同的查询
func (s *HotFeedService) topWindowVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, error) {
	// 第一次检查：有缓存就直接返回
	cachedVideos, hit, err := s.hotVideoRepo.GetCachedWindowVideos(
		ctx,
		limit,
	)
	if err != nil {
		log.Printf("读取热门列表缓存失败，继续查询榜单: %v", err)
	} else if hit { //有没有读到可用缓存
		return cachedVideos, nil
	}

	// 相同 limit 的查询使用相同的合并 key
	key := fmt.Sprintf("hot-window-feed:%d", limit)

	resultCh := s.hotFeedGroup.DoChan(key, func() (any, error) {
		// 为共享查询设置独立的 5 秒期限
		queryCtx, queryCancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			5*time.Second,
		)
		defer queryCancel()

		// 再次检查缓存
		cachedVideos, hit, err := s.hotVideoRepo.GetCachedWindowVideos(
			queryCtx,
			limit,
		)
		if err != nil {
			log.Printf("再次读取热门列表缓存失败，继续查询榜单: %v", err)
		} else if hit {
			return cachedVideos, nil
		}

		// 查询窗口排名和 MySQL 视频详情
		videos, err := s.loadTopVideos(queryCtx, limit)
		if err != nil {
			return nil, err
		}

		// 保存列表缓存
		if err := s.hotVideoRepo.SetCachedWindowVideos(
			queryCtx,
			limit,
			videos,
		); err != nil {
			log.Printf("保存热门列表缓存失败: %v", err)
		}

		return videos, nil
	})
	select {
	case <-ctx.Done():
		// 当前请求取消，结束它自己的等待
		return nil, ctx.Err()

	case result := <-resultCh:
		// 共享查询完成，检查结果
		if result.Err != nil {
			return nil, result.Err
		}

		videos, ok := result.Val.([]model.Video)
		if !ok {
			return nil, fmt.Errorf("热门列表合并查询结果类型错误")
		}

		return videos, nil
	}
}

// 优先返回窗口热榜，窗口榜不可用时尝试最新视频兜底
func (s *HotFeedService) TopVideos(
	ctx context.Context,
	limit int,
) (*model.HotFeedResult, error) {
	videos, err := s.topWindowVideos(ctx, limit)
	if err == nil {
		return &model.HotFeedResult{
			Videos:   videos,
			Degraded: false,
			Source:   model.HotFeedSourceWindow,
		}, nil
	}

	// 当前请求已经取消，就不再启动兜底查询
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// 只有窗口榜不可用，才进入降级路径
	if !errors.Is(err, ErrHotRankingUnavailable) {
		return nil, err
	}

	log.Printf(
		"窗口热榜不可用，尝试最新视频降级：limit=%d，err=%v",
		limit,
		err,
	)

	latestVideos, fallbackErr := s.topLatestVideos(ctx, limit)
	if fallbackErr != nil {
		return nil, fallbackErr
	}

	return &model.HotFeedResult{
		Videos:   latestVideos,
		Degraded: true,
		Source:   model.HotFeedSourceLatest,
	}, nil
}
