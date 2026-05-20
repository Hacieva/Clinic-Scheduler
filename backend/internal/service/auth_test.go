package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type mockUserRepo struct {
	user              *model.User
	err               error
	updatePasswordErr error
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByID(_ context.Context, _ int64) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, _ int64, _ string) error {
	return m.updatePasswordErr
}

func validUser(active bool) *model.User {
	hash, _ := auth.HashPassword("ValidPass1!")
	return &model.User{
		ID:           1,
		Email:        "doctor@clinic.local",
		PasswordHash: hash,
		Role:         model.RoleDoctor,
		IsActive:     active,
	}
}

const testJWTSecret = "test-secret-for-service-tests"

func TestLogin_Success(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	result, err := svc.Login(context.Background(), "doctor@clinic.local", "ValidPass1!")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, int64(1), result.User.ID)

	claims, err := auth.ValidateToken(result.AccessToken, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(1), claims.UserID)
	assert.Equal(t, model.RoleDoctor, claims.Role)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{err: apperrors.ErrNotFound}, testJWTSecret)

	result, err := svc.Login(context.Background(), "nobody@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestLogin_InactiveUser(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(false)}, testJWTSecret)

	result, err := svc.Login(context.Background(), "doctor@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, apperrors.ErrInactiveUser)
	assert.Nil(t, result)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	result, err := svc.Login(context.Background(), "doctor@clinic.local", "WrongPass999!")
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
	assert.Nil(t, result)
}

func TestLogin_EmptyJWTSecret(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, "")

	result, err := svc.Login(context.Background(), "doctor@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, auth.ErrEmptySecret)
	assert.Nil(t, result)
}

// — Refresh —

func TestRefresh_Success(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	token, err := auth.GenerateRefreshToken(1, model.RoleDoctor, testJWTSecret)
	require.NoError(t, err)

	result, err := svc.Refresh(context.Background(), token)
	require.NoError(t, err)
	require.NotNil(t, result)

	accessClaims, err := auth.ValidateToken(result.AccessToken, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "access", accessClaims.TokenType)
	assert.Equal(t, int64(1), accessClaims.UserID)

	refreshClaims, err := auth.ValidateToken(result.RefreshToken, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "refresh", refreshClaims.TokenType)
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{}, testJWTSecret)

	_, err := svc.Refresh(context.Background(), "not.a.valid.token")
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	accessToken, err := auth.GenerateAccessToken(1, model.RoleDoctor, testJWTSecret)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), accessToken)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestRefresh_InactiveUser(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(false)}, testJWTSecret)

	token, err := auth.GenerateRefreshToken(1, model.RoleDoctor, testJWTSecret)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), token)
	assert.ErrorIs(t, err, apperrors.ErrInactiveUser)
}

func TestRefresh_UserNotFound(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{err: apperrors.ErrNotFound}, testJWTSecret)

	token, err := auth.GenerateRefreshToken(1, model.RoleDoctor, testJWTSecret)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), token)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — GetMe —

func TestGetMe_Success(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	user, err := svc.GetMe(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "doctor@clinic.local", user.Email)
}

func TestGetMe_NotFound(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{err: apperrors.ErrNotFound}, testJWTSecret)

	user, err := svc.GetMe(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, user)
}

// — ChangePassword —

func TestChangePassword_Success(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	err := svc.ChangePassword(context.Background(), 1, "ValidPass1!", "NewSecure99!")
	require.NoError(t, err)
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	err := svc.ChangePassword(context.Background(), 1, "WrongPassword!", "NewSecure99!")
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestChangePassword_WeakNew(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{user: validUser(true)}, testJWTSecret)

	err := svc.ChangePassword(context.Background(), 1, "ValidPass1!", "short")
	assert.ErrorIs(t, err, auth.ErrWeakPassword)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{err: apperrors.ErrNotFound}, testJWTSecret)

	err := svc.ChangePassword(context.Background(), 999, "ValidPass1!", "NewSecure99!")
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
