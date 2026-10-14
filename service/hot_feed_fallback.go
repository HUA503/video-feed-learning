package service

import (
	"context"
	"fmt"
	"time"

	"example.com/video-feed/model"
)

// 一份最新视频列表的本地缓存
type latestFeedCacheEntry struct {
	videos    []model.Video
	expiresAt time.Time //列表什么时候过期
}

// 读取未过期的最新视频缓存
func (s *HotFeedService) getCachedLatestVideos(
	limit int,
) ([]model.Video, bool) {
	s.latestCacheMu.RLock()
	defer s.latestCacheMu.RUnlock()

	entry, exists := s.latestCache[limit]
	if !exists || !time.Now().Before(entry.expiresAt) {
		return nil, false
	}

	return entry.videos, true
}

// 保存最新视频缓存，有效期为 5 秒
func (s *HotFeedService) setCachedLatestVideos(
	limit int,
	videos []model.Video,
) {
	if videos == nil {
		videos = make([]model.Video, 0)
	}

	s.latestCacheMu.Lock()
	defer s.latestCacheMu.Unlock()

	s.latestCache[limit] = latestFeedCacheEntry{
		videos:    videos,
		expiresAt: time.Now().Add(5 * time.Second),
	}
}

// 优先使用本地缓存，未命中时合并查询最新视频
func (s *HotFeedService) topLatestVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if videos, hit := s.getCachedLatestVideos(limit); hit {
		return videos, nil
	}

	key := fmt.Sprintf("latest-feed:%d", limit)

	//拿通道
	resultCh := s.hotFeedGroup.DoChan(key, func() (any, error) {
		// 再次检查，其他请求可能已经填好了缓存
		if videos, hit := s.getCachedLatestVideos(limit); hit {
			return videos, nil
		}

		// 为共享的 MySQL 查询设置独立的 2 秒期限
		queryCtx, queryCancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			2*time.Second,
		)
		defer queryCancel()

		//调用 Repository，去 MySQL 查询最新发布的视频
		videos, err := s.videoRepo.ListLatest(queryCtx, limit)
		if err != nil {
			return nil, fmt.Errorf("查询降级最新视频失败: %w", err)
		}

		s.setCachedLatestVideos(limit, videos)

		return videos, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}

		videos, ok := result.Val.([]model.Video)
		if !ok {
			return nil, fmt.Errorf("最新视频合并查询结果类型错误")
		}

		return videos, nil
	}
}
