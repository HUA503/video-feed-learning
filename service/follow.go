package service

import (
	"errors"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var (
	ErrCannotFollowSelf = errors.New("不能关注自己")
	ErrFolloweeNotFound = errors.New("被关注的用户不存在")
	ErrAlreadyFollowing = errors.New("已经关注该用户")
)

type FollowService struct {
	followRepo *repository.FollowRepository
	userRepo   *repository.UserRepository
}

func NewFollowService(
	followRepo *repository.FollowRepository,
	userRepo *repository.UserRepository,
) *FollowService {
	return &FollowService{
		followRepo: followRepo,
		userRepo:   userRepo,
	}
}

func (s *FollowService) Follow(followerID, followeeID uint) error {
	//不能自己关注自己
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}
	//检查被关注者是否存在
	_, err := s.userRepo.FindByID(followeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFolloweeNotFound
		}
		return err
	}
	//检查关注者是否存在
	exists, err := s.followRepo.Exists(followerID, followeeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyFollowing
	}

	follow := &model.Follow{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}

	return s.followRepo.Create(follow)
}
