package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// 创建JWT
// serct 签名的密钥
func GenerateToken(userID uint, secret []byte) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), //过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	// SignedString(secret)为密钥签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret) //用 secret 给令牌签名
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

	return claims, nil
}
