package crypto

import (
	"crypto/ed25519"
	"testing"
)

func TestSignatureVerification(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	message := []byte("Crypto test message")
	sig := ed25519.Sign(priv, message)

	if !ed25519.Verify(pub, message, sig) {
		t.Fatal("signature verification failed")
	}
}
