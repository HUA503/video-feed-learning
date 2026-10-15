package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"example.com/video-feed/auth"
	"example.com/video-feed/repository"
	"github.com/gin-gonic/gin"
)

const (
	UserIDKey         = "userID"
	TokenIDKey        = "tokenID"
	TokenExpiresAtKey = "tokenExpiresAt"
)

// 登录鉴权
// 接收 JWT 密钥
func AuthMiddleware(
	secret []byte,
	tokenRevocationRepo *repository.TokenRevocationRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2) //拆分 Bearer 和令牌

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "请提供有效的登录令牌",
			})
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "请提供有效的登录令牌",
			})
			return
		}

		claims, err := auth.ParseToken(tokenString, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "登录令牌无效或已过期",
			})
			return
		}

		// 最多用 2 秒查询 token 是否已被撤销
		checkCtx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)

		revoked, err := tokenRevocationRepo.IsRevoked(
			checkCtx,
			claims.ID,
		)
		cancel()

		if err != nil {
			log.Printf("查询 token 撤销状态失败: %v", err)

			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "登录状态暂时无法验证，请稍后重试",
			})
			return
		}

		if revoked {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "登录凭证已失效，请重新登录",
			})
			return
		}

		// 全部检查通过后，保存当前请求的身份和 token 信息
		c.Set(UserIDKey, claims.UserID)
		c.Set(TokenIDKey, claims.ID)
		c.Set(TokenExpiresAtKey, claims.ExpiresAt.Time)

		c.Next()
	}
}
