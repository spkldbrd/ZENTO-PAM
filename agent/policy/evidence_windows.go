//go:build windows

package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// TargetEvidence returns SHA-256 (lowercase hex) and best-effort Authenticode publisher for API evidence.
func TargetEvidence(exePath string) (hashHex string, publisher string, err error) {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	hashHex = strings.ToLower(hex.EncodeToString(sum[:]))
	pub, perr := filePublisher(exePath)
	if perr != nil {
		return hashHex, "", nil
	}
	return hashHex, pub, nil
}
