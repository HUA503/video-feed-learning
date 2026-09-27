package service

import (
	"errors"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var ErrAlreadyLiked = errors.New("已经点过赞")

type VideoLikeService struct {
	likeRepo  *repository.VideoLikeRepository //创建点赞记录
	videoRepo *repository.VideoRepository     //查询目标视频是否存在
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
	//保存点赞记录
	return s.likeRepo.Create(like)
}
