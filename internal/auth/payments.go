package auth

import (
	"net/http"
	"strings"
)

func GetAPIKey(header http.Header) string {
	authorizationHeader := header.Get("Authorization")
	if authorizationHeader == "" || !strings.HasPrefix(authorizationHeader, "ApiKey") {
		return ""
	}
	apiKey := strings.TrimPrefix(authorizationHeader, "ApiKey ")
	return apiKey
}
