package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pam-platform/agent/audit"
	"pam-platform/agent/backend"
	"pam-platform/agent/config"
	"pam-platform/agent/ipc"
)

// RunAgent starts audit logging, optional backend registration, and the named pipe server until ctx is done.
func RunAgent(ctx context.Context, baseDir string) error {
	log, err := audit.New(baseDir)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	defer log.Close()

	cfgPath := filepath.Join(baseDir, "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	var bc *backend.Client
	if cfg.BackendEnabled() {
		regCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		c, err := backend.Connect(regCtx, cfg, baseDir)
		cancel()
		if err != nil {
			log.Write(audit.Entry{
				Event:    "registration",
				Mode:     "backend",
				Decision: "denied",
				Reason:   err.Error(),
				Result:   "register_failed",
			})
			bc = nil
		} else {
			bc = c
			if c != nil {
				log.Write(audit.Entry{
					Event:    "registration",
					Mode:     "backend",
					Decision: "allowed",
					Result:   "registered",
					DeviceID: c.DeviceID(),
				})
			}
		}
	}

	srv := &ipc.Server{
		BaseDir: baseDir,
		Log:     log,
		Config:  cfg,
		Backend: bc,
	}
	return srv.ListenAndServe(ctx)
}

// BaseDirFromExecutable returns the directory containing the running binary.
func BaseDirFromExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.Abs(filepath.Clean(exe))
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
