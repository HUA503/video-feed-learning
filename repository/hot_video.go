package repository

import (
	"context"
	"strconv"

	"example.com/video-feed/model"
	"github.com/redis/go-redis/v9"
)

// 热门榜在 Redis 中使用的 key
const hotVideosKey = "hot:videos"

// 检查一条热度事件是否处理过
// 调用时的键列表放进 KEYS，键列表后面的参数依次放进 ARGV
var incrementScoreOnceScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[2]) == 1 then
    return 0
end

redis.call("ZINCRBY", KEYS[1], ARGV[1], ARGV[2])
redis.call("SET", KEYS[2], "1", "EX", ARGV[3])

return 1
`)

type HotVideoRepository struct {
	redisClient *redis.Client
}

func NewHotVideoRepository(redisClient *redis.Client) *HotVideoRepository {
	return &HotVideoRepository{redisClient: redisClient}
}

// 给某个视频在热门榜里增加热度分数
// 把参数传给脚本，再把执行结果返回给调用方
func (r *HotVideoRepository) IncrementScoreOnce(
	ctx context.Context,
	eventID string,
	videoID uint,
	delta float64,
) (bool, error) {
	videoIDText := strconv.FormatUint(uint64(videoID), 10)
	eventKey := "hot:event:" + eventID

	const eventTTLSeconds = 7 * 24 * 60 * 60

	//传入脚本
	result, err := incrementScoreOnceScript.Run(
		ctx,
		r.redisClient,
		[]string{hotVideosKey, eventKey},
		delta,
		videoIDText,
		eventTTLSeconds,
	).Int64()
	if err != nil {
		return false, err
	}

	return result == 1, nil
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

// 根据多个视频 ID 查询详情，并传递上下文
func (r *VideoRepository) FindByIDs(
	ctx context.Context,
	ids []uint, //要查询的视频 ID 切片
) ([]model.Video, error) {
	videos := make([]model.Video, 0)

	if len(ids) == 0 {
		return videos, nil
	}

	err := r.db.
		WithContext(ctx). //把上下文交给 GORM，让这次数据库操作能收到取消或超时通知
		Where("id IN ?", ids).
		Find(&videos).
		Error

	return videos, err
}
