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

// State 返回当前登录用户的关注状态。
func (h *FollowHandler) State(c *gin.Context) {
	followeeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || followeeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 格式不正确"})
		return
	}
	value, exists := c.Get(middleware.UserIDKey)
	followerID, ok := value.(uint)
	if !exists || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到登录用户"})
		return
	}
	following, err := h.followService.State(followerID, uint(followeeID))
	if errors.Is(err, service.ErrFolloweeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if err != nil {
		log.Printf("查询关注状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询关注状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"following": following})
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

// 取消关注
func (h *FollowHandler) Unfollow(c *gin.Context) {
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

	err = h.followService.Unfollow(followerID, uint(followeeIDValue))
	if err != nil {
		if errors.Is(err, service.ErrNotFollowing) {
			c.JSON(http.StatusNotFound, gin.H{"error": "尚未关注该用户"})
			return
		}

		log.Printf("取消关注失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消关注失败"})
		return
	}

	c.Status(http.StatusNoContent)
}
