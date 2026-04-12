package store

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const sessionTokenHashPrefix = "sha256:"

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return sessionTokenHashPrefix + base64.RawURLEncoding.EncodeToString(sum[:])
}
