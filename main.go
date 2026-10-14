package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	// 收到 Ctrl+C 时，取消 appCtx，通知后台任务停止
	appCtx, cancelApp := signal.NotifyContext( //创建一个上下文，并让它监听指定的系统信号
		context.Background(), //创建这套上下文关系的起点
		os.Interrupt,         //监听的中断信号
	)
	defer cancelApp()

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

	// 登记一个消费者任务
	var consumerWG sync.WaitGroup
	consumerWG.Add(1)

	go func() {
		defer consumerWG.Done()

		for {
			// 当前消息处理完后，开始下一轮前先检查停止通知
			if appCtx.Err() != nil {
				log.Println("热度消费者已停止")
				return
			}

			var delivery amqp.Delivery

			select {
			case <-appCtx.Done():
				log.Println("热度消费者已停止")
				return

			case message, ok := <-deliveries:
				if !ok {
					log.Println("热度消息接收通道已关闭，消费者结束")
					return
				}

				delivery = message
			}

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
			var windowUpdated bool
			const maxAttempts = 3

			//尝试三次，测试是否是redis出问题
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				ctx, cancel := context.WithTimeout( //创建上下文，并给这次 Redis 操作设置 5 秒超时
					context.Background(),
					5*time.Second,
				)

				//处理当前热度事件，接收处理结果，然后释放本次上下文的资源
				// 处理累计榜
				updated, err = hotVideoRepo.IncrementScoreOnce(
					ctx,
					event.EventID,
					event.VideoID,
					event.Delta,
				)

				// 累计榜处理成功后，再处理窗口榜
				windowUpdated = false

				if err == nil && !event.OccurredAt.IsZero() {
					windowUpdated, err = hotVideoRepo.RecordWindowEventOnce(
						ctx,
						event.EventID,
						event.VideoID,
						event.Delta,
						event.OccurredAt,
					)
				}

				// 两个 Redis 操作结束后，再释放上下文资源
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
			log.Printf(
				"热度事件处理完成：event_id=%s，video_id=%d，delta=%v，累计榜本次更新=%t，窗口榜本次更新=%t",
				event.EventID,
				event.VideoID,
				event.Delta,
				updated,
				windowUpdated,
			)
		}
	}()

	// 登记窗口热度清理任务
	var windowCleanupWG sync.WaitGroup
	windowCleanupWG.Add(1)

	go func() {
		defer windowCleanupWG.Done()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-appCtx.Done():
				log.Println("窗口热度清理任务已停止")
				return

			case <-ticker.C:
				// 每次清理最多使用 5 秒
				cleanupCtx, cleanupCancel := context.WithTimeout(
					appCtx,
					5*time.Second,
				)

				cleaned, err := hotVideoRepo.CleanupExpiredWindowEvents(
					cleanupCtx,
					100,
				)
				cleanupCancel()

				if err != nil {
					if appCtx.Err() != nil {
						log.Println("窗口热度清理任务已停止")
						return
					}

					log.Printf("清理过期窗口事件失败: %v", err)
					continue
				}

				if cleaned > 0 {
					log.Printf("已清理过期窗口事件：数量=%d", cleaned)
				}
			}
		}
	}()

	log.Println("窗口热度清理任务已启动")

	hotFeedService := service.NewHotFeedService(hotVideoRepo, videoRepo)
	hotFeedHandler := handler.NewHotFeedHandler(hotFeedService)

	outboxRepo := repository.NewHotScoreOutboxRepository(db)
	outboxService := service.NewHotScoreOutboxService(
		outboxRepo,
		hotScoreChannel,
	)

	// 登记一个需要等待的后台任务
	var outboxWG sync.WaitGroup
	outboxWG.Add(1)

	go func() {
		// 这个 goroutine 结束时，通知任务完成
		defer outboxWG.Done()

		ticker := time.NewTicker(2 * time.Second) //定时器
		defer ticker.Stop()

		for {
			select {
			case <-appCtx.Done(): //上下文被取消
				log.Println("Outbox 后台发送任务已停止")
				return

			case <-ticker.C: //定时器触发时
				if err := outboxService.PublishPending(
					appCtx,
					10,
				); err != nil {
					if appCtx.Err() != nil {
						log.Println("Outbox 后台发送任务已停止")
						return
					}

					log.Printf("后台发布 Outbox 事件失败: %v", err)
				}
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

	// 创建 HTTP 服务，使用 Gin 处理请求
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}

	// 在另一个 goroutine 中运行 HTTP 服务
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP 服务运行失败: %v", err)
			cancelApp()
		}
	}()

	// main 在这里等待停止通知
	<-appCtx.Done()
	log.Println("收到停止通知，准备关闭 HTTP 服务")

	// 给正在处理的 HTTP 请求最多 10 秒收尾时间
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP 服务平滑关闭失败: %v", err)

		// 等待失败后，关闭剩余连接
		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("关闭 HTTP 连接失败: %v", closeErr)
		}
	}

	log.Println("HTTP 服务已停止")

	// 三个后台任务共用一个等待期限，最多等待 20 秒
	backgroundStopCtx, backgroundStopCancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer backgroundStopCancel()

	log.Println("等待 Outbox 后台任务结束")
	if err := waitForBackgroundTask(backgroundStopCtx, &outboxWG); err != nil {
		log.Printf("等待 Outbox 后台任务结束失败: %v", err)
	} else {
		log.Println("Outbox 后台任务已结束")
	}

	log.Println("等待热度消费者结束")
	if err := waitForBackgroundTask(backgroundStopCtx, &consumerWG); err != nil {
		log.Printf("等待热度消费者结束失败: %v", err)
	} else {
		log.Println("热度消费者已结束")
	}

	log.Println("等待窗口热度清理任务结束")
	if err := waitForBackgroundTask(backgroundStopCtx, &windowCleanupWG); err != nil {
		log.Printf("等待窗口热度清理任务结束失败: %v", err)
	} else {
		log.Println("窗口热度清理任务已结束")
	}

	log.Println("后台退出等待结束，准备关闭连接并退出")

}

// 等待一个后台任务结束，同时允许上下文取消或超时
func waitForBackgroundTask(
	ctx context.Context, //控制这次等待的期限或取消
	wg *sync.WaitGroup, //指定要等待哪个任务
) error {
	done := make(chan struct{}) //完成通知通道

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
