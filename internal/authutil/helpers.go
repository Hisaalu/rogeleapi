package authutil

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

func GenerateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func Now() time.Time {
	return time.Now().UTC()
}

func DerefOr(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}
