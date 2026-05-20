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
	user *model.User
	err  error
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByID(_ context.Context, _ int64) (*model.User, error) {
	return m.user, m.err
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
