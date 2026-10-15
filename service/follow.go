package service

import (
	"errors"

	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"gorm.io/gorm"
)

var ErrNotFollowing = errors.New("尚未关注该用户")

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

// State 查询当前用户与目标用户之间的关注关系。
func (s *FollowService) State(followerID, followeeID uint) (bool, error) {
	if _, err := s.userRepo.FindByID(followeeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrFolloweeNotFound
		}
		return false, err
	}
	return s.followRepo.Exists(followerID, followeeID)
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

// 取消关注的业务
func (s *FollowService) Unfollow(followerID, followeeID uint) error {
	exists, err := s.followRepo.Exists(followerID, followeeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFollowing
	}

	return s.followRepo.Delete(followerID, followeeID)
}

// 获取关注列表
func (s *FollowService) ListFolloweeIDs(followerID uint) ([]uint, error) {
	return s.followRepo.ListFolloweeIDs(followerID)
}
