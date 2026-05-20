package auth

import (
	"errors"
	"time"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrEmptySecret  = errors.New("jwt secret must not be empty")
)

type Claims struct {
	UserID int64          `json:"user_id"`
	Role   model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID int64, role model.UserRole, secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	return generate(userID, role, secret, accessTokenTTL)
}

func GenerateRefreshToken(userID int64, role model.UserRole, secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	return generate(userID, role, secret, refreshTokenTTL)
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !t.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func generate(userID int64, role model.UserRole, secret string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
