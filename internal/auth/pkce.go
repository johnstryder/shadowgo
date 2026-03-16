package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GeneratePKCE creates a code_verifier and code_challenge for OAuth 2.0 PKCE (S256).
func GeneratePKCE() (verifier, challenge string, err error) {
	// code_verifier: 43-128 chars, base64url
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)

	// code_challenge = BASE64URL(SHA256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}
