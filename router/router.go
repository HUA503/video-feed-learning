package router

import (
	"example.com/video-feed/handler"
	"example.com/video-feed/middleware"
	"example.com/video-feed/repository"
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
	tokenRevocationRepo *repository.TokenRevocationRepository,
	logoutHandler *handler.LogoutHandler,
) *gin.Engine {
	engine := gin.Default()

	// 页面与 API 使用同一个地址，浏览器可以直接调用现有接口。
	engine.Static("/assets", "./web/assets")
	engine.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./web/index.html")
	})
	engine.GET("/healthz", handler.Health)
	engine.POST("/api/v1/users", userHandler.Register)
	engine.POST("/api/v1/login", userHandler.Login)
	engine.POST(
		"/api/v1/logout",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		logoutHandler.Logout,
	)
	engine.GET(
		"/api/v1/me",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		userHandler.Me,
	)

	engine.GET("/api/v1/videos/:id", videoHandler.GetByID)
	engine.POST(
		"/api/v1/videos",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		videoHandler.Create,
	)
	engine.GET("/api/v1/videos", videoHandler.List)
	//点赞
	engine.GET(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		videoLikeHandler.State,
	)
	engine.POST(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		videoLikeHandler.Like,
	)
	//取消点赞
	engine.DELETE(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		videoLikeHandler.Unlike,
	)
	//创见评论
	engine.POST(
		"/api/v1/videos/:id/comments",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
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
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		commentHandler.Delete,
	)

	//关注
	engine.GET(
		"/api/v1/users/:id/follow",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		followHandler.State,
	)
	engine.POST(
		"/api/v1/users/:id/follow",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		followHandler.Follow,
	)

	//取消关注
	engine.DELETE(
		"/api/v1/users/:id/follow",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		followHandler.Unfollow,
	)

	//注册关注feed
	engine.GET(
		"/api/v1/feed/following",
		middleware.AuthMiddleware(jwtSecret, tokenRevocationRepo),
		feedHandler.Following,
	)

	//热门视频
	engine.GET(
		"/api/v1/feed/hot",
		hotFeedHandler.List,
	)
	return engine
}
