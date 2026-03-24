//go:build !windows

package broker

import "fmt"

func Launch(exePath, args, workingDir string) (uint32, error) {
	return 0, fmt.Errorf("broker.Launch only supported on windows")
}
