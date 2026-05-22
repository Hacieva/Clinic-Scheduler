package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func TestGenerateAndValidate_AccessToken(t *testing.T) {
	token, err := GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, model.RoleAdmin, claims.Role)
	assert.Equal(t, "access", claims.TokenType)
	assert.Nil(t, claims.BranchIDs)
}

func TestGenerateAndValidate_RefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken(7, model.RoleDoctor, nil, testSecret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(7), claims.UserID)
	assert.Equal(t, model.RoleDoctor, claims.Role)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestGenerateAccessToken_WithBranchIDs(t *testing.T) {
	ids := []int64{1, 3, 7}
	token, err := GenerateAccessToken(10, model.RoleAdmin, ids, testSecret)
	require.NoError(t, err)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, ids, claims.BranchIDs)
}

func TestGenerateRefreshToken_WithBranchIDs(t *testing.T) {
	ids := []int64{2}
	token, err := GenerateRefreshToken(11, model.RoleDoctor, ids, testSecret)
	require.NoError(t, err)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, ids, claims.BranchIDs)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := GenerateAccessToken(1, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	_, err = ValidateToken(token, "wrong-secret")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_Expired(t *testing.T) {
	claims := &Claims{
		UserID: 1,
		Role:   model.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = ValidateToken(token, testSecret)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_Tampered(t *testing.T) {
	token, err := GenerateAccessToken(1, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	// Flip one character in the signature (last segment)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	sig := []byte(parts[2])
	sig[0] ^= 0x01
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	_, err = ValidateToken(tampered, testSecret)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenTypes_Distinct(t *testing.T) {
	access, err := GenerateAccessToken(1, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)
	refresh, err := GenerateRefreshToken(1, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	accessClaims, err := ValidateToken(access, testSecret)
	require.NoError(t, err)
	refreshClaims, err := ValidateToken(refresh, testSecret)
	require.NoError(t, err)

	assert.Equal(t, "access", accessClaims.TokenType)
	assert.Equal(t, "refresh", refreshClaims.TokenType)
	assert.NotEqual(t, accessClaims.TokenType, refreshClaims.TokenType)
}

func TestGenerateAccessToken_EmptySecret(t *testing.T) {
	_, err := GenerateAccessToken(1, model.RoleAdmin, nil, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}

func TestGenerateRefreshToken_EmptySecret(t *testing.T) {
	_, err := GenerateRefreshToken(1, model.RoleDoctor, nil, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}

func TestValidateToken_EmptySecret(t *testing.T) {
	_, err := ValidateToken("anytoken", "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}
