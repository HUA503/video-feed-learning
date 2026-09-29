package router

import (
	"example.com/video-feed/handler"
	"example.com/video-feed/middleware"
	"github.com/gin-gonic/gin"
)

func New(
	userHandler *handler.UserHandler,
	videoHandler *handler.VideoHandler,
	videoLikeHandler *handler.VideoLikeHandler,
	jwtSecret []byte,
	commentHandler *handler.CommentHandler,
	followHandler *handler.FollowHandler,
	feedHandler *handler.FeedHandler,
	hotFeedHandler *handler.HotFeedHandler,
) *gin.Engine {
	engine := gin.Default()
	engine.GET("/healthz", handler.Health)
	engine.POST("/api/v1/users", userHandler.Register)
	engine.POST("/api/v1/login", userHandler.Login)
	engine.GET(
		"/api/v1/me",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.Me,
	)

	engine.GET("/api/v1/videos/:id", videoHandler.GetByID)
	engine.POST(
		"/api/v1/videos",
		middleware.AuthMiddleware(jwtSecret),
		videoHandler.Create,
	)
	engine.GET("/api/v1/videos", videoHandler.List)
	//点赞
	engine.POST(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret),
		videoLikeHandler.Like,
	)
	//取消点赞
	engine.DELETE(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret),
		videoLikeHandler.Unlike,
	)
	//创见评论
	engine.POST(
		"/api/v1/videos/:id/comments",
		middleware.AuthMiddleware(jwtSecret),
		commentHandler.Create,
	)

	//查询评论
	engine.GET(
		"/api/v1/videos/:id/comments",
		commentHandler.List,
	)

	//删除评论
	engine.DELETE(
		"/api/v1/comments/:id",
		middleware.AuthMiddleware(jwtSecret),
		commentHandler.Delete,
	)

	//关注
	engine.POST(
		"/api/v1/users/:id/follow",
		middleware.AuthMiddleware(jwtSecret),
		followHandler.Follow,
	)

	//取消关注
	engine.DELETE(
		"/api/v1/users/:id/follow",
		middleware.AuthMiddleware(jwtSecret),
		followHandler.Unfollow,
	)

	//注册关注feed
	engine.GET(
		"/api/v1/feed/following",
		middleware.AuthMiddleware(jwtSecret),
		feedHandler.Following,
	)

	//热门视频
	engine.GET(
		"/api/v1/feed/hot",
		hotFeedHandler.List,
	)
	return engine
}
