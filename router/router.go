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

	engine.POST(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret),
		videoLikeHandler.Like,
	)

	engine.DELETE(
		"/api/v1/videos/:id/like",
		middleware.AuthMiddleware(jwtSecret),
		videoLikeHandler.Unlike,
	)
	engine.POST(
		"/api/v1/videos/:id/comments",
		middleware.AuthMiddleware(jwtSecret),
		commentHandler.Create,
	)
	return engine
}
