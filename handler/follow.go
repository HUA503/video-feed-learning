package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"example.com/video-feed/middleware"
	"example.com/video-feed/service"
	"github.com/gin-gonic/gin"
)

type FollowHandler struct {
	followService *service.FollowService
}

func NewFollowHandler(followService *service.FollowService) *FollowHandler {
	return &FollowHandler{followService: followService}
}

func (h *FollowHandler) Follow(c *gin.Context) {
	followeeIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || followeeIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 格式不正确"})
		return
	}

	followerIDValue, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到登录用户"})
		return
	}

	followerID, ok := followerIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录用户信息无效"})
		return
	}

	err = h.followService.Follow(followerID, uint(followeeIDValue))
	if err != nil {
		if errors.Is(err, service.ErrFolloweeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		if errors.Is(err, service.ErrCannotFollowSelf) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能关注自己"})
			return
		}
		if errors.Is(err, service.ErrAlreadyFollowing) {
			c.JSON(http.StatusConflict, gin.H{"error": "已经关注该用户"})
			return
		}

		log.Printf("关注用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "关注失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "关注成功"})
}
