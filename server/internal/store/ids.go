package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewID returns a random 128-bit identifier as 32 hex characters.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// parseNullableTime parses an optional RFC3339Nano timestamp, returning nil for
// a nil pointer or an unparseable value.
func parseNullableTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		return nil
	}
	return &t
}
