package auth

import (
	"strings"
	"testing"
)

const pass = "bad-password"
const expectedHash = "$argon2id$"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, expectedHash) {
		t.Fatalf("expected a different hash, got %s", hash)
	}
}

func TestCheckPasswordHashWithValidPassword(t *testing.T) {
	hashed, err := HashPassword(pass)
	if err != nil {
		t.Fatal(err)
	}
	got, err := CheckPasswordHash(pass, hashed)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected true, got false")
	}
}

func TestCheckPasswordHashWithInvalidPassword(t *testing.T) {
	hashed, err := HashPassword(pass)
	if err != nil {
		t.Fatal(err)
	}
	got, err := CheckPasswordHash("not-the-password", hashed)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("expected false, got true")
	}
}
