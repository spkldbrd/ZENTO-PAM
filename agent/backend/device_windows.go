//go:build windows

package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// DeviceFingerprint returns sha256:hex per API (stable for this machine).
func DeviceFingerprint() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	guid, err := machineGUID()
	if err != nil {
		guid = "unknown"
	}
	h := sha256.Sum256([]byte(strings.ToLower(host) + "\n" + strings.ToLower(guid)))
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

func machineGUID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue("MachineGuid")
	return v, err
}

// WindowsOSInfo fills RegisterOS from registry (best effort).
func WindowsOSInfo() RegisterOS {
	var out RegisterOS
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return out
	}
	defer k.Close()
	if s, _, err := k.GetStringValue("ProductName"); err == nil {
		out.Edition = s
	}
	if s, _, err := k.GetStringValue("DisplayVersion"); err == nil {
		out.Version = s
	}
	if s, _, err := k.GetStringValue("CurrentBuild"); err == nil {
		out.Build = s
	}
	if out.Version == "" && out.Build != "" {
		out.Version = fmt.Sprintf("10.0.%s", out.Build)
	}
	return out
}
