package handler

import (
	"log"
	"net/http"
	"strconv"

	"example.com/video-feed/service"
	"github.com/gin-gonic/gin"
)

type HotFeedHandler struct {
	hotFeedService *service.HotFeedService
}

func NewHotFeedHandler(hotFeedService *service.HotFeedService) *HotFeedHandler {
	return &HotFeedHandler{hotFeedService: hotFeedService}
}

func (h *HotFeedHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit 必须是 1 到 50 之间的数字"})
		return
	}

	videos, err := h.hotFeedService.TopVideos(c.Request.Context(), limit)
	if err != nil {
		log.Printf("查询热门视频失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询热门视频失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videos,
		"limit":  limit,
	})
}
