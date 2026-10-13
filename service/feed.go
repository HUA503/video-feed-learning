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

// 使用游标读取关注 Feed，并判断是否还有下一页
func (s *FeedService) FollowingFeedAfter(
	userID uint,
	cursor *model.VideoCursor,
	pageSize int,
) ([]model.Video, bool, error) {
	if pageSize < 1 || pageSize > maxVideoPageSize {
		return nil, false, ErrInvalidPagination
	}

	// 查询当前用户关注的作者 ID
	followeeIDs, err := s.followService.ListFolloweeIDs(userID)
	if err != nil {
		return nil, false, err
	}

	// 多查一条，用于判断是否还有下一页
	videos, err := s.videoRepo.ListByUserIDsAfter(
		followeeIDs,
		cursor,
		pageSize+1,
	)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(videos) > pageSize

	// 多出来的一条只用于判断，不返回给本页
	if hasMore {
		videos = videos[:pageSize]
	}

	return videos, hasMore, nil
}
