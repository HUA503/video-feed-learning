package repository

import (
	"context"
	"strconv"

	"example.com/video-feed/model"
	"github.com/redis/go-redis/v9"
)

// 热门榜在 Redis 中使用的 key
const hotVideosKey = "hot:videos"

type HotVideoRepository struct {
	redisClient *redis.Client
}

func NewHotVideoRepository(redisClient *redis.Client) *HotVideoRepository {
	return &HotVideoRepository{redisClient: redisClient}
}

// 给某个视频在热门榜里增加热度分数
func (r *HotVideoRepository) IncrementScore(
	ctx context.Context, //调用方传入的上下文，用于控制这次 Redis 操作
	videoID uint,
	delta float64, //本次增加的分数，点赞加 1、评论加 3
) error {
	videoIDText := strconv.FormatUint(uint64(videoID), 10)

	return r.redisClient.
		ZIncrBy(ctx, hotVideosKey, delta, videoIDText).
		Err()
}

// 读取热度最高的视频 ID
func (r *HotVideoRepository) TopVideoIDs(
	ctx context.Context,
	limit int, //读取范围就是 Redis 排名第 0 到第 9 位，也就是前 10 个视频
) ([]uint, error) {
	videoIDs := make([]uint, 0)

	if limit <= 0 {
		return videoIDs, nil
	}

	videoIDTexts, err := r.redisClient.
		ZRevRange(ctx, hotVideosKey, 0, int64(limit-1)).
		Result()
	if err != nil {
		return nil, err
	}

	//逐个处理 Redis 返回的视频 ID 字符串
	for _, videoIDText := range videoIDTexts {
		id, err := strconv.ParseUint(videoIDText, 10, 64)
		if err != nil {
			return nil, err
		}

		videoIDs = append(videoIDs, uint(id))
	}

	return videoIDs, nil
}

// 用上面筛选的 ID 去 MySQL 查询视频详情
func (r *VideoRepository) FindByIDs(ids []uint) ([]model.Video, error) {
	videos := make([]model.Video, 0)

	if len(ids) == 0 {
		return videos, nil
	}

	err := r.db.
		Where("id IN ?", ids).
		Find(&videos).
		Error

	return videos, err
}
