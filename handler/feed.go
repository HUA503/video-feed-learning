package handler

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/video-feed/middleware"
	"example.com/video-feed/service"
	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	feedService *service.FeedService
}

func NewFeedHandler(feedService *service.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

func (h *FeedHandler) Following(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到登录用户"})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录用户信息无效"})
		return
	}

	//从 URL 查询参数中读取页码，并把它从字符串转换成整数
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page 必须是数字"})
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page_size 必须是数字"})
		return
	}

	videos, total, err := h.feedService.FollowingFeed(userID, page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPagination) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分页参数不合法"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询关注 Feed 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos":    videos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
