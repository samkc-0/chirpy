package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	s := hex.EncodeToString(b)
	return s, nil
}
