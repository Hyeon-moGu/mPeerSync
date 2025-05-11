package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"

	"mPeerSync/util"
)

func GenerateAndSaveKeyPair() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := SaveKeyPair(priv, pub); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("keys", "my_public.key"), pub, 0644); err != nil {
		return err
	}
	util.Success("Key pair generated and saved. Public key saved to keys/my_public.key")
	return nil
}

func SaveKeyPair(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) error {
	if err := os.WriteFile("private.key", privateKey, 0600); err != nil {
		return err
	}
	if err := os.WriteFile("public.key", publicKey, 0644); err != nil {
		return err
	}
	return nil
}

func LoadKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privKeyBytes, err := os.ReadFile("private.key")
	if err != nil {
		return nil, nil, err
	}
	pubKeyBytes, err := os.ReadFile("public.key")
	if err != nil {
		return nil, nil, err
	}
	return ed25519.PublicKey(pubKeyBytes), ed25519.PrivateKey(privKeyBytes), nil
}

func LoadKeyring(dir string) ([]ed25519.PublicKey, []string, error) {
	var keys []ed25519.PublicKey
	var filenames []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			util.Warn("Failed to load key %s: %v", f.Name(), err)
			continue
		}
		keys = append(keys, ed25519.PublicKey(data))
		filenames = append(filenames, f.Name())
	}
	return keys, filenames, nil
}

func DecodeHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}
