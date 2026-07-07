package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// NewToken returns a URL-safe random token of n bytes (~1.3*n chars after b64).
// Used for session cookies and one-shot QR login tokens.
func NewToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
