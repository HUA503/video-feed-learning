package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// 创建带随机编号的 JWT
func GenerateToken(userID uint, secret []byte) (string, error) {
	tokenIDBytes := make([]byte, 16) //生成 16 字节的安全随机数据
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", fmt.Errorf("生成令牌 ID 失败: %w", err)
	}

	now := time.Now()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        hex.EncodeToString(tokenIDBytes),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// JWT 解析与验证
func ParseToken(tokenString string, secret []byte) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString, //要解析的令牌
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %v",
					token.Header["alg"],
				)
			}
			return secret, nil
		},
	)
	if err != nil {
		return nil, err
	}
	//检查令牌对象的有效状态
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// 必须包含有效的用户 ID
	if claims.UserID == 0 {
		return nil, errors.New("令牌缺少有效的用户 ID")
	}

	// 必须包含 token 的独立编号
	if claims.ID == "" {
		return nil, errors.New("令牌缺少编号")
	}

	// 必须包含过期时间
	if claims.ExpiresAt == nil {
		return nil, errors.New("令牌缺少过期时间")
	}

	return claims, nil
}
