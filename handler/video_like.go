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

type VideoLikeHandler struct {
	likeService *service.VideoLikeService
}

func NewVideoLikeHandler(likeService *service.VideoLikeService) *VideoLikeHandler {
	return &VideoLikeHandler{likeService: likeService}
}

// State 返回当前登录用户的点赞状态。
func (h *VideoLikeHandler) State(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || videoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视频 ID 格式不正确"})
		return
	}
	value, exists := c.Get(middleware.UserIDKey)
	userID, ok := value.(uint)
	if !exists || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到登录用户"})
		return
	}
	liked, err := h.likeService.State(userID, uint(videoID))
	if errors.Is(err, service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}
	if err != nil {
		log.Printf("查询点赞状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询点赞状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

// 取 URL 里的视频 ID，从鉴权中间件中取得当前用户 ID
func (h *VideoLikeHandler) Like(c *gin.Context) {
	videoIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "视频 ID 格式不正确",
		})
		return
	}

	userIDValue, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未获取到登录用户",
		})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "登录用户信息无效",
		})
		return
	}

	err = h.likeService.Like(userID, uint(videoIDValue))
	if err != nil {
		if errors.Is(err, service.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "视频不存在",
			})
			return
		}

		if errors.Is(err, service.ErrAlreadyLiked) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "已经点过赞",
			})
			return
		}

		log.Printf("视频点赞失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "点赞失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "点赞成功",
	})
}

// 取消点赞
func (h *VideoLikeHandler) Unlike(c *gin.Context) {
	videoIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "视频 ID 格式不正确",
		})
		return
	}

	userIDValue, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未获取到登录用户",
		})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "登录用户信息无效",
		})
		return
	}

	if err := h.likeService.Unlike(userID, uint(videoIDValue)); err != nil {
		if errors.Is(err, service.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "视频不存在",
			})
			return
		}

		log.Printf("取消视频点赞失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "取消点赞失败",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
