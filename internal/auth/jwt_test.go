package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTPipeline(t *testing.T) {
	userID := uuid.New()
	ss, err := MakeJWT(userID, "secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsedUserID, err := ValidateJWT(ss, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if parsedUserID != userID {
		t.Fatal("parsed user id does not match original user id")
	}
}

func TestInvalidSecretFailsToValidate(t *testing.T) {
	userID := uuid.New()
	ss, err := MakeJWT(userID, "secret", 1*time.Minute)
	_, err = ValidateJWT(ss, "notthesecret")
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}
}

func TestTokenDoesNotParseIfExpired(t *testing.T) {
	userID := uuid.New()
	ss, err := MakeJWT(userID, "secret", 1*time.Second)
	time.Sleep(2 * time.Second)
	_, err = ValidateJWT(ss, "secret")
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	header := &http.Header{}
	want := "abcd123"
	header.Set("Authorization", "Bearer "+want)
	got := GetBearerToken(*header)
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
