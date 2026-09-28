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

// 把 URL 中的视频 ID 交给 Service，再把评论列表作为 HTTP 响应返回
func (h *CommentHandler) List(c *gin.Context) {
	videoIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || videoIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视频 ID 格式不正确"})
		return
	}

	comments, err := h.commentService.ListByVideoID(uint(videoIDValue))
	if err != nil {
		if errors.Is(err, service.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
			return
		}

		log.Printf("查询评论列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询评论失败"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// 删除评论
func (h *CommentHandler) Delete(c *gin.Context) {
	commentIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || commentIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论 ID 格式不正确"})
		return
	}

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

	err = h.commentService.Delete(userID, uint(commentIDValue))
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
			return
		}
		if errors.Is(err, service.ErrNotCommentOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己的评论"})
			return
		}

		log.Printf("删除评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}

	c.Status(http.StatusNoContent)
}
