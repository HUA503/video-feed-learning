package handler

import (
	"log"
	"net/http"

	"example.com/video-feed/middleware"
	"example.com/video-feed/service"
	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	videoService *service.VideoService
}

func NewVideoHandler(videoService *service.VideoService) *VideoHandler {
	return &VideoHandler{videoService: videoService}
}

type createVideoRequest struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"omitempty,max=5000"`
	VideoURL    string `json:"video_url" binding:"required,max=500"`
	CoverURL    string `json:"cover_url" binding:"omitempty,max=500"`
}

func (h *VideoHandler) Create(c *gin.Context) {
	var req createVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "视频信息格式不正确",
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

	video, err := h.videoService.Create(
		userID,
		req.Title,
		req.Description,
		req.VideoURL,
		req.CoverURL,
	)
	if err != nil {
		log.Printf("创建视频失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "视频发布失败",
		})
		return
	}

	c.JSON(http.StatusCreated, video)
}
