package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_Valid(t *testing.T) {
	hash, err := HashPassword("securepass")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := HashPassword("short")
	assert.ErrorIs(t, err, ErrWeakPassword)
}

func TestVerifyPassword_Match(t *testing.T) {
	hash, err := HashPassword("securepass")
	require.NoError(t, err)
	assert.NoError(t, VerifyPassword(hash, "securepass"))
}

func TestVerifyPassword_Mismatch(t *testing.T) {
	hash, err := HashPassword("securepass")
	require.NoError(t, err)
	assert.Error(t, VerifyPassword(hash, "wrongpass"))
}

func TestValidatePasswordStrength_Edge(t *testing.T) {
	assert.NoError(t, ValidatePasswordStrength("exactly8"))
}

func TestValidatePasswordStrength_SpacesOnlyOrTrimmedShort(t *testing.T) {
	cases := []string{
		"        ",    // 8 spaces only
		"   hi   ",   // trimmed = "hi" (2 chars)
		"  short ",   // trimmed = "short" (5 chars)
		" 1234567",   // trimmed = "1234567" (7 chars)
	}
	for _, plain := range cases {
		assert.ErrorIs(t, ValidatePasswordStrength(plain), ErrWeakPassword, "input: %q", plain)
	}
}
