package service

import (
	"errors"
	"strings"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var ErrInvalidComment = errors.New("评论内容不能为空")

type CommentService struct {
	commentRepo *repository.CommentRepository
	videoRepo   *repository.VideoRepository
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	videoRepo *repository.VideoRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		videoRepo:   videoRepo,
	}
}

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
