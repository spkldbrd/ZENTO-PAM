//go:build windows

package broker

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Launch starts exePath with optional args and workingDir using a duplicated primary token
// from the current process (LocalSystem when running as a service) and CreateProcessAsUser.
// (CreateProcessWithTokenW is equivalent for a primary token; x/sys provides CreateProcessAsUser.)
// The child runs in session 0 service context, not the interactive user desktop (Phase 1 limitation).
func Launch(exePath, args, workingDir string) (pid uint32, err error) {
	exePath = filepath.Clean(exePath)
	cmdLine, err := windows.UTF16PtrFromString(buildCommandLine(exePath, args))
	if err != nil {
		return 0, err
	}

	var appName *uint16
	var cwd *uint16
	if workingDir != "" {
		cwd, err = windows.UTF16PtrFromString(workingDir)
		if err != nil {
			return 0, err
		}
	}

	var self windows.Token
	err = windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_QUERY, &self)
	if err != nil {
		return 0, fmt.Errorf("OpenProcessToken: %w", err)
	}
	defer self.Close()

	var dup windows.Token
	err = windows.DuplicateTokenEx(self, windows.MAXIMUM_ALLOWED, nil, windows.SecurityImpersonation, windows.TokenPrimary, &dup)
	if err != nil {
		return 0, fmt.Errorf("DuplicateTokenEx: %w", err)
	}
	defer dup.Close()

	si := &windows.StartupInfo{}
	si.Cb = uint32(unsafe.Sizeof(*si))
	pi := &windows.ProcessInformation{}

	const createFlags = windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NEW_CONSOLE

	err = windows.CreateProcessAsUser(dup, appName, cmdLine, nil, nil, false, createFlags, nil, cwd, si, pi)
	if err != nil {
		return 0, fmt.Errorf("CreateProcessAsUser: %w", err)
	}
	windows.CloseHandle(pi.Thread)
	windows.CloseHandle(pi.Process)
	return pi.ProcessId, nil
}

func buildCommandLine(exePath, args string) string {
	exe := exePath
	if strings.ContainsAny(exe, ` `+"\t") {
		exe = `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	}
	if strings.TrimSpace(args) == "" {
		return exe
	}
	return exe + " " + args
}
