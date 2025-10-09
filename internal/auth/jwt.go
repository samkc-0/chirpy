package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

func MakeJWT(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	issuedAt := jwt.NewNumericDate(now)
	expiresAt := jwt.NewNumericDate(now.Add(expiresIn))
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return signedString, nil
}

func ValidateJWT(tokenString string, secret string) (uuid.UUID, error) {
	parser := func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, parser)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("token string is invalid or expired: %s", err)
	}
	subject, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("token was parsed, but failed to get subject: %s", err)
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse uuid from string to uuid.UUID: %s", err)
	}
	return userID, nil
}

func GetBearerToken(header http.Header) string {
	authorizationHeader := header.Get("Authorization")
	tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
	return tokenString
}
