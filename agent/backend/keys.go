package backend

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
)

// LoadOrGenerateEd25519 loads PEM-less raw keys from data/ed25519.key or creates them.
func LoadOrGenerateEd25519(baseDir string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	p := keysPath(baseDir)
	b, err := os.ReadFile(p)
	if err == nil && len(b) > 0 {
		priv := ed25519.PrivateKey(b)
		if len(priv) != ed25519.PrivateKeySize {
			return nil, nil, os.ErrInvalid
		}
		return priv, priv.Public().(ed25519.PublicKey), nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return nil, nil, err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(priv), 0600); err != nil {
		return nil, nil, err
	}
	if err := os.Rename(tmp, p); err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// PublicKeyBase64 returns API-shaped public key material.
func PublicKeyBase64(pub ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub)
}
