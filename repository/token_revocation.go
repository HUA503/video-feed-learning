package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenRevocationRepository struct {
	redisClient *redis.Client
}

func NewTokenRevocationRepository(
	redisClient *redis.Client,
) *TokenRevocationRepository {
	return &TokenRevocationRepository{
		redisClient: redisClient,
	}
}

func revokedTokenKey(tokenID string) string {
	return "auth:revoked:" + tokenID
}

// 保存失效记录，到 token 原本的过期时间自动删除
func (r *TokenRevocationRepository) Revoke(
	ctx context.Context,
	tokenID string,
	expiresAt time.Time, //过期时间
) error {
	if tokenID == "" {
		return fmt.Errorf("令牌 ID 不能为空")
	}
	if expiresAt.IsZero() {
		return fmt.Errorf("令牌过期时间不能为空")
	}

	// 已经过期的 token，无需再保存失效记录
	if !expiresAt.After(time.Now()) {
		return nil
	}

	err := r.redisClient.SetArgs(
		ctx,
		revokedTokenKey(tokenID),
		"1",
		redis.SetArgs{
			ExpireAt: expiresAt,
		},
	).Err()
	if err != nil {
		return fmt.Errorf("保存令牌失效记录失败: %w", err)
	}

	return nil
}

// 判断 token 是否已被撤销
func (r *TokenRevocationRepository) IsRevoked(
	ctx context.Context,
	tokenID string,
) (bool, error) {
	if tokenID == "" {
		return false, fmt.Errorf("令牌 ID 不能为空")
	}

	count, err := r.redisClient.Exists(
		ctx,
		revokedTokenKey(tokenID),
	).Result()
	if err != nil {
		return false, fmt.Errorf("查询令牌失效记录失败: %w", err)
	}

	return count > 0, nil
}
