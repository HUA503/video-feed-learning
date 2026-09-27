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

// 从 URL 里取出视频 ID，调用 Service 查数据库，并根据结果返回对应的 HTTP 状态码
func (h *VideoHandler) GetByID(c *gin.Context) {
	idValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "视频 ID 格式不正确",
		})
		return
	}

	video, err := h.videoService.GetByID(uint(idValue))
	if err != nil {
		if errors.Is(err, service.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "视频不存在",
			})
			return
		}

		log.Printf("查询视频失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询视频失败",
		})
		return
	}

	c.JSON(http.StatusOK, video)
}
