//go:build windows

package policy

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// publisherFromPowerShell returns the Authenticode simple name via Get-AuthenticodeSignature.
// Used as a last resort for catalog-signed OS binaries where CryptQueryObject / WinVerifyTrustEx
// file verification does not surface an embedded PKCS#7.
func publisherFromPowerShell(path string) (string, error) {
	lit := strings.ReplaceAll(path, "'", "''")
	// SimpleName matches CERT_NAME_SIMPLE_DISPLAY_TYPE style used elsewhere.
	ps := fmt.Sprintf(
		`$s = Get-AuthenticodeSignature -LiteralPath '%s'; if ($null -eq $s.SignerCertificate) { exit 2 }; $s.SignerCertificate.GetNameInfo('SimpleName',$false)`,
		lit,
	)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	timer := time.AfterFunc(15*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	err := cmd.Run()
	timer.Stop()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 2 {
			return "", fmt.Errorf("no signer certificate")
		}
		msg := strings.TrimSpace(errOut.String())
		if msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	name := strings.TrimSpace(out.String())
	if name == "" {
		return "", fmt.Errorf("empty publisher from PowerShell")
	}
	return name, nil
}
