//go:build !windows

package policy

import "fmt"

func filePublisher(path string) (string, error) {
	return "", fmt.Errorf("authenticode only supported on windows (got path %s)", path)
}
