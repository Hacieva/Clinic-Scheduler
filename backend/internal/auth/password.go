package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var ErrWeakPassword = errors.New("password must be at least 8 characters")

func HashPassword(plain string) (string, error) {
	if err := ValidatePasswordStrength(plain); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

func ValidatePasswordStrength(plain string) error {
	if len(strings.TrimSpace(plain)) < 8 {
		return ErrWeakPassword
	}
	return nil
}
