package service

import (
	"errors"
	"log"

	"example.com/video-feed/model"
	"example.com/video-feed/queue"
	"example.com/video-feed/repository"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

var ErrAlreadyLiked = errors.New("已经点过赞")

type VideoLikeService struct {
	likeRepo        *repository.VideoLikeRepository //创建点赞记录
	videoRepo       *repository.VideoRepository     //查询目标视频是否存在
	hotScoreChannel *amqp.Channel                   //发布热度消息
}

func NewVideoLikeService(
	likeRepo *repository.VideoLikeRepository,
	videoRepo *repository.VideoRepository,
	hotScoreChannel *amqp.Channel,
) *VideoLikeService { //构造函数接收两个 Repository，创建 VideoLikeService
	return &VideoLikeService{
		likeRepo:        likeRepo,
		videoRepo:       videoRepo,
		hotScoreChannel: hotScoreChannel,
	}
}

// 确认视频存在，再创建点赞记录
func (s *VideoLikeService) Like(userID, videoID uint) error {
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		return err
	}

	like := &model.VideoLike{
		UserID:  userID,
		VideoID: videoID,
	}

	//重复点赞检测
	alreadyLiked, err := s.likeRepo.Exists(userID, videoID)
	if err != nil {
		return err
	}
	if alreadyLiked {
		return ErrAlreadyLiked
	}
	// 先保存点赞记录
	if err := s.likeRepo.Create(like); err != nil {
		return err
	}

	// 保存成功后，发布热度增加事件
	if err := queue.PublishHotScore(
		s.hotScoreChannel,
		queue.HotScoreEvent{
			VideoID: videoID,
			Delta:   1,
		},
	); err != nil {
		log.Printf(
			"点赞已保存，但发布热度事件失败：video_id=%d，err=%v",
			videoID,
			err,
		)
	}

	return nil
}

// 取消点赞
func (s *VideoLikeService) Unlike(userID, videoID uint) error {
	_, err := s.videoRepo.FindByID(videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		return err
	}

	//deleted 用来判断是否真的删了点赞记录
	deleted, err := s.likeRepo.DeleteByUserAndVideo(userID, videoID)
	if err != nil {
		return err
	}

	if !deleted {
		return nil
	}

	if err := queue.PublishHotScore(
		s.hotScoreChannel,
		queue.HotScoreEvent{
			VideoID: videoID,
			Delta:   -1,
		},
	); err != nil {
		log.Printf(
			"点赞已取消，但发布减分事件失败：video_id=%d，err=%v",
			videoID,
			err,
		)
	}

	return nil
}
