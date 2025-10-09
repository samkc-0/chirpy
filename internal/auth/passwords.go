package auth

import (
	"errors"
	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hashed_password, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", errors.New("failed to hash password with argon2id")
	}
	return hashed_password, nil
}

func CheckPasswordHash(password string, hashed_password string) (bool, error) {
	ok, err := argon2id.ComparePasswordAndHash(password, hashed_password)
	if err != nil {
		return false, errors.New("failed to check password with argon2id")
	}
	return ok, nil
}
