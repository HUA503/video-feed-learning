package service

import (
	"example.com/video-feed/model"
	"example.com/video-feed/repository"
)

type FeedService struct {
	followService *FollowService
	videoRepo     *repository.VideoRepository
}

func NewFeedService(
	followService *FollowService,
	videoRepo *repository.VideoRepository,
) *FeedService {
	return &FeedService{
		followService: followService,
		videoRepo:     videoRepo,
	}
}

// 调用 FollowService 和 VideoRepository。
func (s *FeedService) FollowingFeed(
	userID uint,
	page int,
	pageSize int,
) ([]model.Video, int64, error) {
	if page < 1 || pageSize < 1 || pageSize > maxVideoPageSize { //检查分页参数是否合法
		return nil, 0, ErrInvalidPagination
	}

	//查出当前用户关注的所有用户 ID
	followeeIDs, err := s.followService.ListFolloweeIDs(userID)
	if err != nil {
		return nil, 0, err
	}

	return s.videoRepo.ListByUserIDs(followeeIDs, page, pageSize)
}
