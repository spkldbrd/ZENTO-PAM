//go:build !windows

package policy

import "fmt"

func TargetEvidence(exePath string) (string, string, error) {
	return "", "", fmt.Errorf("TargetEvidence: windows only (%s)", exePath)
}
