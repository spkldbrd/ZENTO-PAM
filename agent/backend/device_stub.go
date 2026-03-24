//go:build !windows

package backend

import "os"

func DeviceFingerprint() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return "sha256:nonwindows-" + host, nil
}

func WindowsOSInfo() RegisterOS {
	return RegisterOS{Edition: "non-windows", Version: "0", Build: "0"}
}
