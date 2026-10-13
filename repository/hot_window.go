package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 窗口内各视频的热度分数
	hotWindowVideosKey = "hot:window:videos" //谁更热门

	// 按发生时间排列的事件 ID，用于找出过期事件
	hotWindowEventsKey = "hot:window:events" //哪些贡献已经过期

	// 保存每条事件对应的视频 ID 和 Delta
	hotWindowEventDataKey = "hot:window:event-data" //过期时撤掉多少分

	// 统计最近 24 小时
	hotWindowDuration = 24 * time.Hour
)

var recordWindowEventOnceScript = redis.NewScript(`
local eventID = ARGV[1]
local videoID = ARGV[2]
local delta = tonumber(ARGV[3])
local occurredAt = tonumber(ARGV[4])
local now = tonumber(ARGV[5])
local window = tonumber(ARGV[6])  

-- 事件已经超出窗口，不计入窗口榜
if occurredAt <= now - window then
    return -1
end

-- 事件发生时间不能晚于当前时间
if occurredAt > now then
    return redis.error_reply("event time is in the future")
end

-- 写入前检查三个 key 的数据类型
local videosType = redis.call("TYPE", KEYS[1]).ok
local eventsType = redis.call("TYPE", KEYS[2]).ok
local dataType = redis.call("TYPE", KEYS[3]).ok

if (videosType ~= "none" and videosType ~= "zset")
    or (eventsType ~= "none" and eventsType ~= "zset")
    or (dataType ~= "none" and dataType ~= "hash") then
    return redis.error_reply("window key type mismatch")
end

-- 同一事件已经记录过，不重复计分
if redis.call("HEXISTS", KEYS[3], eventID) == 1 then
    return 0
end

local eventData = cjson.encode({
    video_id = videoID,
    delta = delta
})

-- 更新视频窗口分数
redis.call("ZINCRBY", KEYS[1], delta, videoID)

-- 保存事件发生时间
redis.call("ZADD", KEYS[2], occurredAt, eventID)

-- 保存事件内容，供去重和过期清理使用
redis.call("HSET", KEYS[3], eventID, eventData)

return 1
`)

// 将一条热度事件记录到窗口榜，同一事件只记录一次
func (r *HotVideoRepository) RecordWindowEventOnce(
	ctx context.Context,
	eventID string,
	videoID uint,
	delta float64,
	occurredAt time.Time,
) (bool, error) {
	if eventID == "" || videoID == 0 || delta == 0 {
		return false, fmt.Errorf("窗口热度事件参数不合法")
	}

	if occurredAt.IsZero() {
		return false, fmt.Errorf("窗口热度事件缺少发生时间")
	}

	videoIDText := strconv.FormatUint(uint64(videoID), 10)

	//执行 Lua 脚本
	result, err := recordWindowEventOnceScript.Run(
		ctx,
		r.redisClient,
		[]string{
			hotWindowVideosKey,
			hotWindowEventsKey,
			hotWindowEventDataKey,
		},
		eventID,
		videoIDText,
		delta,
		occurredAt.UnixMilli(),
		time.Now().UnixMilli(),
		hotWindowDuration.Milliseconds(), //窗口长度
	).Int64()

	if err != nil {
		return false, fmt.Errorf("记录窗口热度事件失败: %w", err)
	}

	switch result {
	case 1:
		return true, nil
	case 0, -1:
		return false, nil
	default:
		return false, fmt.Errorf("窗口脚本返回未知结果: %d", result)
	}
}
