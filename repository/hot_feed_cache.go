package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"example.com/video-feed/model"
	"github.com/redis/go-redis/v9"
)

// 不同的 limit 使用不同的缓存 key
func hotFeedCacheKey(limit int) string {
	return fmt.Sprintf("hot:window:feed:cache:v1:%d", limit)
}

// 读取已经组装好的热门视频列表
func (r *HotVideoRepository) GetCachedWindowVideos(
	ctx context.Context,
	limit int,
) ([]model.Video, bool, error) {
	key := hotFeedCacheKey(limit)

	// 从 Redis 读取视频列表的 JSON 数据
	body, err := r.redisClient.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		// key 不存在或已经过期，属于正常的缓存未命中
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("读取热门列表缓存失败: %w", err)
	}

	var videos []model.Video
	if err := json.Unmarshal(body, &videos); err != nil {
		return nil, false, fmt.Errorf("解析热门列表缓存失败: %w", err)
	}

	return videos, true, nil
}

// 保存热门视频列表，缓存有效期为 5 秒
func (r *HotVideoRepository) SetCachedWindowVideos(
	ctx context.Context,
	limit int,
	videos []model.Video,
) error {
	// 空列表统一保存为 []，避免编码成 null
	if videos == nil {
		videos = make([]model.Video, 0)
	}

	body, err := json.Marshal(videos)
	if err != nil {
		return fmt.Errorf("编码热门列表缓存失败: %w", err)
	}

	err = r.redisClient.Set(
		ctx,
		hotFeedCacheKey(limit),
		body,
		5*time.Second,
	).Err()
	if err != nil {
		return fmt.Errorf("保存热门列表缓存失败: %w", err)
	}

	return nil
}
