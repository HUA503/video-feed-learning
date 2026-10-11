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
	amqp "github.com/rabbitmq/amqp091-go"
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

	// 注册 RabbitMQ 连接状态变化通知
	rabbitStates := make(chan *amqp.StateChanged, 8)
	rabbitConn.NotifyStateChange(rabbitStates)

	// 持续接收状态变化
	go func() {
		for change := range rabbitStates {
			log.Printf(
				"RabbitMQ 连接状态变化：%s -> %s",
				change.From,
				change.To,
			)

			if change.Err != nil {
				log.Printf("RabbitMQ 恢复终止：%v", change.Err)
			}

			for _, entity := range change.SkippedTopologyEntities {
				log.Printf("RabbitMQ 恢复时跳过资源：%v", entity)
			}
		}
	}()

	log.Println("RabbitMQ 连接成功")

	//初始化队列    //点赞、评论发布消息
	hotScoreChannel, err := queue.DeclareHotScoreQueue(rabbitConn)
	if err != nil {
		log.Fatalf("初始化热度队列失败: %v", err)
	}
	defer hotScoreChannel.Close()

	//接收消息、确认处理完成
	hotScoreConsumerChannel, err := rabbitConn.Channel()
	if err != nil {
		log.Fatalf("创建热度消费者通道失败: %v", err)
	}
	defer hotScoreConsumerChannel.Close()

	//等待 RabbitMQ 回复“确认模式已开启”
	if err := hotScoreChannel.Confirm(false); err != nil {
		log.Fatalf("开启 RabbitMQ 发送确认模式失败: %v", err)
	}

	log.Println("RabbitMQ 发送确认模式已开启")

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
		&model.HotScoreOutbox{},
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
	)
	videoLikeHandler := handler.NewVideoLikeHandler(videoLikeService)

	//评论
	commentRepo := repository.NewCommentRepository(db)
	commentService := service.NewCommentService(
		commentRepo,
		videoRepo,
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
	deliveries, err := queue.ConsumeHotScore(hotScoreConsumerChannel)
	if err != nil {
		log.Fatalf("启动热度消费者失败: %v", err)
	}

	go func() {
		for delivery := range deliveries {
			event, err := queue.DecodeHotScore(delivery.Body)
			if err != nil {
				log.Printf(
					"收到无效热度消息：err=%v，body=%s",
					err,
					delivery.Body,
				)

				if rejectErr := delivery.Reject(false); rejectErr != nil {
					log.Printf(
						"拒绝无效热度消息失败，继续等待后续投递：%v",
						rejectErr,
					)
					continue
				}

				continue
			}

			//调用之前写好的 Redis 方法
			var updated bool
			const maxAttempts = 3

			//尝试三次，测试是否是redis出问题
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				ctx, cancel := context.WithTimeout( //创建上下文，并给这次 Redis 操作设置 5 秒超时
					context.Background(),
					5*time.Second,
				)

				//处理当前热度事件，接收处理结果，然后释放本次上下文的资源
				updated, err = hotVideoRepo.IncrementScoreOnce(
					ctx,
					event.EventID,
					event.VideoID,
					event.Delta,
				)
				cancel()

				// 成功处理，包括识别出重复事件
				if err == nil {
					break
				}

				log.Printf(
					"更新 Redis 热度失败：event_id=%s，尝试=%d/%d，err=%v",
					event.EventID,
					attempt,
					maxAttempts,
					err,
				)

				// 第一次失败等 1 秒，第二次失败等 2 秒
				if attempt < maxAttempts {
					time.Sleep(time.Duration(attempt) * time.Second)
				}
			}

			// 所有尝试都失败，交给死信队列
			if err != nil {
				log.Printf(
					"热度事件多次处理失败，准备拒绝消息：event_id=%s",
					event.EventID,
				)

				if rejectErr := delivery.Reject(false); rejectErr != nil {
					log.Printf(
						"拒绝热度消息失败，继续等待后续投递：event_id=%s，err=%v",
						event.EventID,
						rejectErr,
					)
					continue
				}

				log.Printf(
					"已发送热度消息拒绝请求：event_id=%s",
					event.EventID,
				)
				continue
			}

			if err := delivery.Ack(false); err != nil {
				log.Printf(
					"确认热度消息失败，继续等待后续投递：event_id=%s，err=%v",
					event.EventID,
					err,
				)
				continue
			}

			// 新增：重复事件已经处理过，跳过后面的成功日志
			if !updated {
				log.Printf("重复热度事件已跳过：event_id=%s", event.EventID)
				continue
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

	outboxRepo := repository.NewHotScoreOutboxRepository(db)
	outboxService := service.NewHotScoreOutboxService(
		outboxRepo,
		hotScoreChannel,
	)

	go func() {
		//创建一个每隔两秒发送一次信号的定时器
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		//定时器的通道
		for range ticker.C {
			if err := outboxService.PublishPending(
				context.Background(),
				10,
			); err != nil {
				log.Printf("后台发布 Outbox 事件失败: %v", err)
			}
		}
	}()

	log.Println("Outbox 后台发送任务已启动")

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
