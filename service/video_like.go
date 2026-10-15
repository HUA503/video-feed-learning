package service

import (
	"errors"

	"example.com/video-feed/model"
	"example.com/video-feed/queue"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var ErrAlreadyLiked = errors.New("已经点过赞")

type VideoLikeService struct {
	likeRepo  *repository.VideoLikeRepository //创建点赞记录
	videoRepo *repository.VideoRepository     //查询目标视频是否存在 	//发布热度消息
}

func NewVideoLikeService(
	likeRepo *repository.VideoLikeRepository,
	videoRepo *repository.VideoRepository,
) *VideoLikeService { //构造函数接收两个 Repository，创建 VideoLikeService
	return &VideoLikeService{
		likeRepo:  likeRepo,
		videoRepo: videoRepo,
	}
}

// State 查询当前用户是否已经点赞，供页面恢复按钮状态。
func (s *VideoLikeService) State(userID, videoID uint) (bool, error) {
	if _, err := s.videoRepo.FindByID(videoID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	return s.likeRepo.Exists(userID, videoID)
}

// 确认视频存在，再创建点赞记录
func (s *VideoLikeService) Like(userID, videoID uint) error {
	// 1. 确认视频存在
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		return err
	}

	// 2. 检查是否已经点赞
	alreadyLiked, err := s.likeRepo.Exists(userID, videoID)
	if err != nil {
		return err
	}
	if alreadyLiked {
		return ErrAlreadyLiked
	}

	// 3. 准备点赞记录
	like := &model.VideoLike{
		UserID:  userID,
		VideoID: videoID,
	}

	// 4. 生成本次点赞事件的 ID
	eventID, err := queue.GenerateEventID()
	if err != nil {
		return err
	}

	// 5. 准备 +1 热度事件
	outboxEvent := &model.HotScoreOutbox{
		EventID: eventID,
		VideoID: videoID,
		Delta:   1,
	}

	// 6. 在同一个事务中保存点赞记录和事件
	if err := s.likeRepo.CreateWithOutbox(
		like,
		outboxEvent,
	); err != nil {
		return err
	}

	return nil
}

// 取消点赞，同时保存减分事件
func (s *VideoLikeService) Unlike(userID, videoID uint) error {
	// 1. 确认视频存在
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		return err
	}

	// 2. 生成本次取消点赞事件的 ID
	eventID, err := queue.GenerateEventID()
	if err != nil {
		return err
	}

	// 3. 准备 -1 热度事件
	outboxEvent := &model.HotScoreOutbox{
		EventID: eventID,
		VideoID: videoID,
		Delta:   -1,
	}

	// 4. 在同一个事务中删除点赞、保存减分事件
	deleted, err := s.likeRepo.DeleteWithOutbox(
		userID,
		videoID,
		outboxEvent,
	)
	if err != nil {
		return err
	}

	// 5. 没有点赞记录，也按取消成功处理
	if !deleted {
		return nil
	}

	return nil
}
