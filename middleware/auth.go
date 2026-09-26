package middleware

import (
	"net/http"
	"strings"

	"example.com/video-feed/auth"
	"github.com/gin-gonic/gin"
)

const UserIDKey = "userID"

// 登录鉴权
// 接收 JWT 密钥
func AuthMiddleware(secret []byte) gin.HandlerFunc {
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

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}
