package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"example.com/video-feed/middleware"
	"example.com/video-feed/model"
	"example.com/video-feed/pagination"
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
	// 1. 获取当前登录用户
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

	// 2. 读取每页条数
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page_size 必须是数字"})
		return
	}

	// 3. 解码请求中的游标；没有游标时读取第一页
	cursor, err := pagination.DecodeVideoCursor(c.Query("cursor"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分页游标不合法"})
		return
	}

	// 4. 查询本页视频和是否还有下一页
	videos, hasMore, err := h.feedService.FollowingFeedAfter(
		userID,
		cursor,
		pageSize,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPagination) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分页参数不合法"})
			return
		}

		log.Printf("查询关注 Feed 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询关注 Feed 失败"})
		return
	}

	// 5. 有下一页时，用本页最后一条视频生成下一页游标
	nextCursor := ""
	if hasMore {
		//索引
		lastVideo := videos[len(videos)-1]

		nextCursor, err = pagination.EncodeVideoCursor(model.VideoCursor{
			CreatedAt: lastVideo.CreatedAt,
			ID:        lastVideo.ID,
		})
		if err != nil {
			log.Printf("生成下一页游标失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成分页游标失败"})
			return
		}
	}

	// 6. 返回本页结果
	c.JSON(http.StatusOK, gin.H{
		"videos":      videos,
		"page_size":   pageSize,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
}
