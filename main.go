package main

import (
	"context"
	"log"
	"os"
	"time"

	"example.com/video-feed/cache"
	"example.com/video-feed/database"
	"example.com/video-feed/handler"
	"example.com/video-feed/model"
	"example.com/video-feed/queue"
	"example.com/video-feed/repository"
	"example.com/video-feed/router"
	"example.com/video-feed/service"
)

func main() {
	//连接 RabbitMQ
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		log.Fatal("RABBITMQ_URL 环境变量未设置")
	}

	rabbitConn, err := queue.ConnectRabbitMQ(rabbitMQURL)
	if err != nil {
		log.Fatalf("连接 RabbitMQ 失败: %v", err)
	}
	defer rabbitConn.Close()

	log.Println("RabbitMQ 连接成功")

	//初始化队列
	hotScoreChannel, err := queue.DeclareHotScoreQueue(rabbitConn)
	if err != nil {
		log.Fatalf("初始化热度队列失败: %v", err)
	}
	defer hotScoreChannel.Close()

	log.Println("RabbitMQ 热度队列已就绪")

	//读取密钥并检查
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET 环境变量未设置")
	}

	//mysql初始化
	db, err := database.ConnectMySQL()
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接池失败: %v", err)
	}
	defer sqlDB.Close()

	log.Println("MySQL 连接成功")

	//redis初始化
	redisClient, err := cache.ConnectRedis()
	if err != nil {
		log.Fatalf("连接 Redis 失败: %v", err)
	}
	defer redisClient.Close()

	log.Println("Redis 连接成功")

	if err := db.AutoMigrate(
		&model.User{},
		&model.Video{},
		&model.VideoLike{},
		&model.Comment{},
		&model.Follow{},
	); err != nil {
		log.Fatalf("数据库表迁移失败: %v", err)
	}
	log.Println("用户表结构已就绪")

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, []byte(jwtSecret))
	userHandler := handler.NewUserHandler(userService)

	//创建并连接视频各层//
	videoRepo := repository.NewVideoRepository(db)
	videoService := service.NewVideoService(videoRepo)
	videoHandler := handler.NewVideoHandler(videoService)

	//点赞
	videoLikeRepo := repository.NewVideoLikeRepository(db)
	videoLikeService := service.NewVideoLikeService(
		videoLikeRepo,
		videoRepo,
		hotScoreChannel,
	)
	videoLikeHandler := handler.NewVideoLikeHandler(videoLikeService)

	//评论
	commentRepo := repository.NewCommentRepository(db)
	commentService := service.NewCommentService(
		commentRepo,
		videoRepo,
		hotScoreChannel,
	)
	commentHandler := handler.NewCommentHandler(commentService)

	//关注（关注塔菲谢谢喵）
	followRepo := repository.NewFollowRepository(db)
	followService := service.NewFollowService(followRepo, userRepo)
	followHandler := handler.NewFollowHandler(followService)

	//feed依赖
	feedService := service.NewFeedService(followService, videoRepo)
	feedHandler := handler.NewFeedHandler(feedService)

	//热门视频
	hotVideoRepo := repository.NewHotVideoRepository(redisClient)

	//验证程序能收到消息，并解析出视频 ID 和分数变化
	deliveries, err := queue.ConsumeHotScore(hotScoreChannel)
	if err != nil {
		log.Fatalf("启动热度消费者失败: %v", err)
	}

	go func() {
		for delivery := range deliveries {
			event, err := queue.DecodeHotScore(delivery.Body)
			if err != nil {
				log.Printf("解析热度消息失败，停止消费: %v", err)
				return
			}

			//创建上下文，并给这次 Redis 操作设置 5 秒超时
			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

			//调用之前写好的 Redis 方法
			err = hotVideoRepo.IncrementScore(
				ctx, // 控制操作的超时、取消
				event.VideoID,
				event.Delta,
			)
			cancel()

			if err != nil {
				log.Printf("更新 Redis 热度失败，停止消费: %v", err)
				return
			}

			if err := delivery.Ack(false); err != nil {
				log.Printf("确认热度消息失败，停止消费: %v", err)
				return
			}

			log.Printf(
				"热度更新完成：video_id=%d，delta=%v",
				event.VideoID,
				event.Delta,
			)
		}
	}()

	hotFeedService := service.NewHotFeedService(hotVideoRepo, videoRepo)
	hotFeedHandler := handler.NewHotFeedHandler(hotFeedService)

	engine := router.New(
		userHandler,
		videoHandler,
		videoLikeHandler,
		[]byte(jwtSecret),
		commentHandler,
		followHandler,
		feedHandler,
		hotFeedHandler,
	)

	if err := engine.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
