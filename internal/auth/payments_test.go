package auth

import (
	"net/http"
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	testHeader := http.Header{}
	testHeader.Set("Authorization", "ApiKey THE_KEY_HERE")
	got := GetAPIKey(testHeader)
	if got != "THE_KEY_HERE" {
		t.Fatalf("got %s, want THE_KEY_HERE", got)
	}
}
