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

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

type createCommentRequest struct {
	Content string `json:"content" binding:"required,max=2000"`
}

func (h *CommentHandler) Create(c *gin.Context) {
	videoIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || videoIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视频 ID 格式不正确"})
		return
	}
	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "评论格式不正确",
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
	comment, err := h.commentService.Create(
		userID,
		uint(videoIDValue),
		req.Content,
	)
	if err != nil {
		if errors.Is(err, service.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
			return
		}
		if errors.Is(err, service.ErrInvalidComment) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "评论内容不能为空"})
			return
		}

		log.Printf("创建评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论发布失败"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}
