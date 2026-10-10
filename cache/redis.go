package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,

		ContextTimeoutEnabled: true, //控制等待
		MaxRetries:            -1,   //让重试次数由我们写的消费者代码管理
	})
	//连接检查设了 5 秒时限
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
