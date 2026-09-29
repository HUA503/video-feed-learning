package main

import (
	"log"
	"os"

	"example.com/video-feed/database"
	"example.com/video-feed/handler"
	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"example.com/video-feed/router"
	"example.com/video-feed/service"
)

func main() {
	//读取密钥并检查
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET 环境变量未设置")
	}

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
	videoLikeService := service.NewVideoLikeService(videoLikeRepo, videoRepo)
	videoLikeHandler := handler.NewVideoLikeHandler(videoLikeService)

	//评论
	commentRepo := repository.NewCommentRepository(db)
	commentService := service.NewCommentService(commentRepo, videoRepo)
	commentHandler := handler.NewCommentHandler(commentService)

	//关注（关注塔菲谢谢喵）
	followRepo := repository.NewFollowRepository(db)
	followService := service.NewFollowService(followRepo, userRepo)
	followHandler := handler.NewFollowHandler(followService)

	//feed依赖
	feedService := service.NewFeedService(followService, videoRepo)
	feedHandler := handler.NewFeedHandler(feedService)

	engine := router.New(
		userHandler,
		videoHandler,
		videoLikeHandler,
		[]byte(jwtSecret),
		commentHandler,
		followHandler,
		feedHandler,
	)

	if err := engine.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
