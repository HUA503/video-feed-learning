package handler

import (
	"log"
	"net/http"
	"time"

	"example.com/video-feed/middleware"
	"example.com/video-feed/service"
	"github.com/gin-gonic/gin"
)

type LogoutHandler struct {
	logoutService *service.LogoutService
}

func NewLogoutHandler(
	logoutService *service.LogoutService,
) *LogoutHandler {
	return &LogoutHandler{
		logoutService: logoutService,
	}
}

func (h *LogoutHandler) Logout(c *gin.Context) {
	// 读取中间件保存的 token 编号
	tokenID := c.GetString(middleware.TokenIDKey)
	if tokenID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未获取到登录令牌信息",
		})
		return
	}

	// 读取中间件保存的过期时间
	expiresAtValue, exists := c.Get(middleware.TokenExpiresAtKey)
	//检查这个值是不是 time.Time，如果是，就放进 expiresAt
	expiresAt, ok := expiresAtValue.(time.Time)

	if !exists || !ok || expiresAt.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "登录令牌过期时间无效",
		})
		return
	}

	// 调用 Service 撤销当前 token
	if err := h.logoutService.Logout(
		//当前 HTTP 请求的上下文
		c.Request.Context(),
		tokenID,
		expiresAt,
	); err != nil {
		log.Printf("退出登录失败: %v", err)

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "退出登录暂时失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "退出登录成功",
	})
}
