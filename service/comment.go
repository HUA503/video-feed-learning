package service

import (
	"errors"
	"strings"

	"example.com/video-feed/model"
	"example.com/video-feed/queue"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var (
	ErrInvalidComment  = errors.New("评论内容不能为空")
	ErrCommentNotFound = errors.New("评论不存在")
	ErrNotCommentOwner = errors.New("无权删除这条评论")
)

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

// 载入评论
func (s *CommentService) Create(
	userID uint,
	videoID uint,
	content string,
) (*model.Comment, error) {
	// 确认视频存在
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	// 检查评论内容
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrInvalidComment
	}

	// 准备评论记录
	comment := &model.Comment{
		UserID:  userID,
		VideoID: videoID,
		Content: content,
	}

	// 在保存事件之前生成固定的事件 ID
	eventID, err := queue.GenerateEventID()
	if err != nil {
		return nil, err
	}

	// 准备待发送的热度事件记录
	outboxEvent := &model.HotScoreOutbox{
		EventID: eventID,
		VideoID: videoID,
		Delta:   3,
	}

	// 在同一个事务中保存评论和事件
	if err := s.commentRepo.CreateWithOutbox(
		comment,
		outboxEvent,
	); err != nil {
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
	// 1. 查询要删除的评论
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	// 2. 只有评论作者才能删除
	if comment.UserID != userID {
		return ErrNotCommentOwner
	}

	// 3. 为本次删除生成一个新的事件 ID
	eventID, err := queue.GenerateEventID()
	if err != nil {
		return err
	}

	// 4. 准备减分事件
	outboxEvent := &model.HotScoreOutbox{
		EventID: eventID,
		VideoID: comment.VideoID,
		Delta:   -3,
	}

	// 5. 在同一个事务中删除评论、保存减分事件
	deleted, err := s.commentRepo.DeleteWithOutbox(
		commentID,
		outboxEvent,
	)
	if err != nil {
		return err
	}

	if !deleted {
		return ErrCommentNotFound
	}

	return nil
}
