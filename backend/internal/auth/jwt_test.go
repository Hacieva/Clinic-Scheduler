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
	token, err := GenerateAccessToken(42, model.RoleAdmin, testSecret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, model.RoleAdmin, claims.Role)
	assert.Equal(t, "access", claims.TokenType)
}

func TestGenerateAndValidate_RefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken(7, model.RoleDoctor, testSecret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(7), claims.UserID)
	assert.Equal(t, model.RoleDoctor, claims.Role)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := GenerateAccessToken(1, model.RoleAdmin, testSecret)
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
	token, err := GenerateAccessToken(1, model.RoleAdmin, testSecret)
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
	access, err := GenerateAccessToken(1, model.RoleAdmin, testSecret)
	require.NoError(t, err)
	refresh, err := GenerateRefreshToken(1, model.RoleAdmin, testSecret)
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
	_, err := GenerateAccessToken(1, model.RoleAdmin, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}

func TestGenerateRefreshToken_EmptySecret(t *testing.T) {
	_, err := GenerateRefreshToken(1, model.RoleDoctor, "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}

func TestValidateToken_EmptySecret(t *testing.T) {
	_, err := ValidateToken("anytoken", "")
	assert.ErrorIs(t, err, ErrEmptySecret)
}
