package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGeneratePKCE_ShouldProduceValidVerifierAndChallenge(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}

	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length should be 43-128, got %d", len(verifier))
	}

	hash := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])
	if challenge != expected {
		t.Errorf("challenge mismatch: got %s", challenge)
	}
}
