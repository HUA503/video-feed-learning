package service

import (
	"context"
	"time"

	"example.com/video-feed/repository"
)

type LogoutService struct {
	tokenRevocationRepo *repository.TokenRevocationRepository
}

func NewLogoutService(
	tokenRevocationRepo *repository.TokenRevocationRepository,
) *LogoutService {
	return &LogoutService{
		tokenRevocationRepo: tokenRevocationRepo,
	}
}

// 撤销当前 token
func (s *LogoutService) Logout(
	ctx context.Context,
	tokenID string,
	expiresAt time.Time,
) error {
	revokeCtx, cancel := context.WithTimeout(
		ctx,
		2*time.Second,
	)
	defer cancel()

	return s.tokenRevocationRepo.Revoke(
		revokeCtx,
		tokenID,
		expiresAt,
	)
}
