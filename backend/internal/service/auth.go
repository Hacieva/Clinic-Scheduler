package service

import (
	"context"
	"log/slog"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	User         *model.User
}

type AuthService struct {
	users     repository.UserRepository
	jwtSecret string
}

func NewAuthService(users repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		slog.WarnContext(ctx, "login: user not found", "email", email)
		return nil, err
	}

	if !user.IsActive {
		slog.WarnContext(ctx, "login: inactive user", "user_id", user.ID)
		return nil, apperrors.ErrInactiveUser
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		slog.WarnContext(ctx, "login: wrong password", "user_id", user.ID)
		return nil, apperrors.ErrUnauthorized
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "login: success", "user_id", user.ID, "role", user.Role)
	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}
