package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// StreamKeyPrefix makes stream keys recognizable, e.g. in secret scanners.
const StreamKeyPrefix = "sk_"

// NewToken returns a random URL-safe token and its SHA-256 hex digest.
// Store only the digest; the token is shown to its owner once.
func NewToken(prefix string) (token, hash string) {
	b := make([]byte, 32)
	rand.Read(b) // never fails; panics if the OS has no randomness
	token = prefix + base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TokenMatches(token, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashToken(token)), []byte(hash)) == 1
}
