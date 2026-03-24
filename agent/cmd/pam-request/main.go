//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"pam-platform/agent/ipc"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <exe_path> [args...]\n", os.Args[0])
		os.Exit(2)
	}
	exePath := os.Args[1]
	var args string
	if len(os.Args) > 2 {
		args = strings.Join(os.Args[2:], " ")
	}

	sid, err := currentUserSID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not get user SID: %v\n", err)
	}

	username := ""
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	req := ipc.Request{
		ExePath:    exePath,
		Args:       args,
		WorkingDir: "",
		UserSID:    sid,
		Username:   username,
		ParentPID:  int32(os.Getppid()),
		ParentPath: "",
	}

	payload, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	conn, err := winio.DialPipe(ipc.PipeName, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if _, err := conn.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	respBody, err := io.ReadAll(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	respBody = bytes.TrimSpace(respBody)

	var resp ipc.Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "response: %v (raw: %s)\n", err, string(respBody))
		os.Exit(1)
	}

	if !resp.OK {
		if resp.Error != "" {
			fmt.Fprintln(os.Stderr, resp.Error)
		} else {
			fmt.Fprintln(os.Stderr, "request denied")
		}
		os.Exit(1)
	}
	fmt.Printf("ok pid=%d\n", resp.PID)
}

func currentUserSID() (string, error) {
	tok := windows.GetCurrentProcessToken()
	tu, err := tok.GetTokenUser()
	if err != nil {
		return "", err
	}
	return tu.User.Sid.String(), nil
}
