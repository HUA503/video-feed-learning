package service

import (
	"errors"
	"strings"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

var (
	ErrInvalidComment  = errors.New("评论内容不能为空")
	ErrCommentNotFound = errors.New("评论不存在")
	ErrNotCommentOwner = errors.New("无权删除这条评论")
)

type CommentService struct {
	commentRepo     *repository.CommentRepository
	videoRepo       *repository.VideoRepository
	hotScoreChannel *amqp.Channel
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	videoRepo *repository.VideoRepository,
	hotScoreChannel *amqp.Channel,
) *CommentService {
	return &CommentService{
		commentRepo:     commentRepo,
		videoRepo:       videoRepo,
		hotScoreChannel: hotScoreChannel,
	}
}

// 载入评论
func (s *CommentService) Create(
	userID uint,
	videoID uint,
	content string,
) (*model.Comment, error) {
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrInvalidComment
	}

	comment := &model.Comment{
		UserID:  userID,
		VideoID: videoID,
		Content: content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// 组织业务流程：先确认视频确实存在，再查询这个视频的评论
func (s *CommentService) ListByVideoID(videoID uint) ([]model.Comment, error) {
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	return s.commentRepo.FindByVideoID(videoID)
}

// 先按评论 ID查评论，比较评论的 UserID 和当前登录用户的 ID
func (s *CommentService) Delete(userID, commentID uint) error {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	if comment.UserID != userID {
		return ErrNotCommentOwner
	}

	return s.commentRepo.DeleteByID(commentID)
}
