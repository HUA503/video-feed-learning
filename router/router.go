package router

import (
	"example.com/video-feed/handler"
	"example.com/video-feed/middleware"
	"github.com/gin-gonic/gin"
)

func New(userHandler *handler.UserHandler, jwtSecret []byte) *gin.Engine {
	engine := gin.Default()
	engine.GET("/healthz", handler.Health)
	engine.POST("/api/v1/users", userHandler.Register)
	engine.POST("/api/v1/login", userHandler.Login)
	engine.GET(
		"/api/v1/me",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.Me,
	)
	return engine
}
