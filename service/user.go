package service

import (
	"errors"

	"example.com/video-feed/auth"
	"example.com/video-feed/model"
	"example.com/video-feed/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("用户名或密码错误")

// 保存密钥并签发令牌
type UserService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte //接收并保存密钥
}

func NewUserService(userRepo *repository.UserRepository, jwtSecret []byte) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// 处理密码并创建用户记录
func (s *UserService) Register(username, password string) (*model.User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// 登录校验
func (s *UserService) Login(username, password string) (*model.User, string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}
	//校验密码
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}
