package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pam-platform/agent/audit"
	"pam-platform/agent/backend"
	"pam-platform/agent/broker"
	"pam-platform/agent/config"
	"pam-platform/agent/policy"

	winio "github.com/Microsoft/go-winio"
)

const PipeName = `\\.\pipe\pam_elevation`

// PipeSDDL: SY/BA full control; Authenticated Users read/write/connect-style access for local clients.
const PipeSDDL = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;AU)"

const maxRequestBytes = 256 * 1024

// Server hosts the elevation named pipe and dispatches requests.
type Server struct {
	BaseDir string
	Log     *audit.Logger
	Config  *config.Config
	Backend *backend.Client
}

// ListenAndServe accepts connections until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	cfg := &winio.PipeConfig{
		SecurityDescriptor: PipeSDDL,
	}
	ln, err := winio.ListenPipe(PipeName, cfg)
	if err != nil {
		return fmt.Errorf("ListenPipe: %w", err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) cfg() *config.Config {
	if s.Config != nil {
		return s.Config
	}
	c := &config.Config{}
	c.Default()
	return c
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	dec := json.NewDecoder(io.LimitReader(conn, maxRequestBytes))
	var req Request
	if err := dec.Decode(&req); err != nil {
		_ = writeResponse(conn, Response{OK: false, Error: "json: " + err.Error()})
		return
	}

	cfg := s.cfg()

	exePath := filepath.Clean(req.ExePath)
	if exePath == "" || exePath == "." {
		wdEarly := strings.TrimSpace(req.WorkingDir)
		e := s.commonElevAudit(req, req.ExePath, wdEarly, "")
		e.Decision, e.Reason, e.Result, e.Mode = "denied", "empty exe_path", "skipped", modeLabel(cfg)
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: "exe_path required"})
		return
	}
	if !filepath.IsAbs(exePath) {
		var absErr error
		exePath, absErr = filepath.Abs(exePath)
		if absErr != nil {
			wdEarly := strings.TrimSpace(req.WorkingDir)
			e := s.commonElevAudit(req, req.ExePath, wdEarly, "")
			e.Decision, e.Reason, e.Result, e.Mode = "denied", "abs path: "+absErr.Error(), "skipped", modeLabel(cfg)
			s.writeAudit(e)
			_ = writeResponse(conn, Response{OK: false, Error: absErr.Error()})
			return
		}
	}

	wd := strings.TrimSpace(req.WorkingDir)
	if wd == "" {
		wd = filepath.Dir(exePath)
	}

	st, err := os.Stat(exePath)
	if err != nil || st.IsDir() {
		reason := "exe not found"
		if err != nil {
			reason = err.Error()
		}
		e := s.commonElevAudit(req, exePath, wd, "")
		e.Decision, e.Reason, e.Result, e.Mode = "denied", reason, "skipped", modeLabel(cfg)
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: reason})
		return
	}

	user := strings.TrimSpace(req.Username)
	if user == "" {
		user = `UNKNOWN\user`
	}

	fileSHA, publisher, _ := policy.TargetEvidence(exePath)

	// Strict mode: backend configured but registration failed and no local fallback.
	if cfg.BackendEnabled() && s.Backend == nil && !cfg.LocalFallback {
		e := s.commonElevAudit(req, exePath, wd, fileSHA)
		e.Decision, e.Reason, e.Result, e.Mode = "denied", "backend unavailable (not registered)", "skipped", "backend_required"
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: "backend unavailable: agent not registered and local_fallback is false"})
		return
	}

	localViaFallback := false
	tryBackend := cfg.BackendEnabled() && s.Backend != nil
	if tryBackend {
		corr := backend.NewCorrelationID()
		payload := &backend.ElevationRequestPayload{
			User:      user,
			ExePath:   exePath,
			Hash:      fileSHA,
			Publisher: publisher,
		}

		timeout := time.Duration(cfg.RequestTimeoutSeconds)*time.Second + 60*time.Second
		reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
		out, berr := s.Backend.SubmitAndWait(reqCtx, payload)
		cancel()

		if berr == nil {
			switch strings.ToUpper(out.Status) {
			case "ALLOWED":
				if !backend.GrantAllowsPathAndHash(out.Grant, exePath, fileSHA) {
					e := s.commonElevAudit(req, exePath, wd, fileSHA)
					e.Decision, e.Reason, e.Result, e.Mode = "denied", "backend grant constraints mismatch", "skipped", "backend"
					e.BackendRequestID, e.BackendStatus, e.CorrelationID = out.RequestID, out.Status, corr
					s.writeAudit(e)
					_ = writeResponse(conn, Response{OK: false, Error: "denied: backend grant does not match target", RequestID: out.RequestID, BackendStatus: out.Status})
					return
				}
				pid, lerr := broker.Launch(exePath, req.Args, wd)
				if lerr != nil {
					e := s.commonElevAudit(req, exePath, wd, fileSHA)
					e.Decision, e.Reason, e.Result, e.Mode = "allowed", "backend ALLOWED", "error: "+lerr.Error(), "backend"
					e.BackendRequestID, e.BackendStatus, e.CorrelationID = out.RequestID, "ALLOWED", corr
					s.writeAudit(e)
					_ = writeResponse(conn, Response{OK: false, Error: lerr.Error(), RequestID: out.RequestID, BackendStatus: "ALLOWED"})
					return
				}
				e := s.commonElevAudit(req, exePath, wd, fileSHA)
				e.Decision, e.Reason, e.Result, e.Mode = "allowed", "backend ALLOWED", fmt.Sprintf("launched pid=%d", pid), "backend"
				e.BackendRequestID, e.BackendStatus, e.CorrelationID = out.RequestID, "ALLOWED", corr
				s.writeAudit(e)
				_ = writeResponse(conn, Response{OK: true, PID: pid, RequestID: out.RequestID, BackendStatus: "ALLOWED"})
				return

			case "DENIED", "EXPIRED", "CANCELLED":
				e := s.commonElevAudit(req, exePath, wd, fileSHA)
				e.Decision, e.Reason, e.Result, e.Mode = "denied", "backend "+out.Status, "skipped", "backend"
				e.BackendRequestID, e.BackendStatus, e.CorrelationID = out.RequestID, out.Status, corr
				s.writeAudit(e)
				_ = writeResponse(conn, Response{
					OK: false, Error: fmt.Sprintf("denied: backend status %s", out.Status),
					RequestID: out.RequestID, BackendStatus: out.Status,
				})
				return

			case "TIMEOUT":
				berr = fmt.Errorf("approval timeout")
			default:
				e := s.commonElevAudit(req, exePath, wd, fileSHA)
				e.Decision, e.Reason, e.Result, e.Mode = "denied", "unexpected backend status "+out.Status, "skipped", "backend"
				e.BackendRequestID, e.BackendStatus, e.CorrelationID = out.RequestID, out.Status, corr
				s.writeAudit(e)
				_ = writeResponse(conn, Response{OK: false, Error: "backend: unexpected status " + out.Status, RequestID: out.RequestID, BackendStatus: out.Status})
				return
			}
		}

		if berr != nil && cfg.LocalFallback {
			e := s.commonElevAudit(req, exePath, wd, fileSHA)
			e.Decision, e.Reason, e.Result, e.Mode = "pending", "backend: "+berr.Error(), "local_fallback", "local_fallback"
			e.CorrelationID = corr
			s.writeAudit(e)
			localViaFallback = true
		} else if berr != nil {
			e := s.commonElevAudit(req, exePath, wd, fileSHA)
			e.Decision, e.Reason, e.Result, e.Mode = "denied", "backend: "+berr.Error(), "skipped", "backend"
			e.CorrelationID = corr
			s.writeAudit(e)
			_ = writeResponse(conn, Response{OK: false, Error: "backend: " + berr.Error()})
			return
		}
	}

	// Local-only or local_fallback path
	s.handleLocalPolicy(conn, cfg, req, exePath, wd, fileSHA, localViaFallback)
}

func (s *Server) handleLocalPolicy(conn net.Conn, cfg *config.Config, req Request, exePath, wd, fileSHA string, fromBackendFallback bool) {
	polPath := filepath.Join(s.BaseDir, "policy", "policy.json")
	pol, err := policy.Load(polPath)
	if err != nil {
		e := s.commonElevAudit(req, exePath, wd, fileSHA)
		e.Decision, e.Reason, e.Result, e.Mode = "denied", "policy load: "+err.Error(), "skipped", localMode(cfg, fromBackendFallback)
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: "policy: " + err.Error()})
		return
	}

	res := pol.Evaluate(exePath)
	if !res.Allowed {
		e := s.commonElevAudit(req, exePath, wd, res.HashHex)
		e.Decision, e.Reason, e.Result, e.Mode = "denied", res.Reason, "skipped", localMode(cfg, fromBackendFallback)
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: "denied: " + res.Reason})
		return
	}

	pid, err := broker.Launch(exePath, req.Args, wd)
	if err != nil {
		e := s.commonElevAudit(req, exePath, wd, res.HashHex)
		e.Decision, e.Reason, e.Result, e.Mode = "allowed", res.Reason, "error: "+err.Error(), localMode(cfg, fromBackendFallback)
		s.writeAudit(e)
		_ = writeResponse(conn, Response{OK: false, Error: err.Error()})
		return
	}

	e := s.commonElevAudit(req, exePath, wd, res.HashHex)
	e.Decision, e.Reason, e.Result, e.Mode = "allowed", res.Reason, fmt.Sprintf("launched pid=%d", pid), localMode(cfg, fromBackendFallback)
	s.writeAudit(e)
	_ = writeResponse(conn, Response{OK: true, PID: pid})
}

func modeLabel(cfg *config.Config) string {
	if cfg == nil || !cfg.BackendEnabled() {
		return "local_only"
	}
	if cfg.LocalFallback {
		return "local_fallback"
	}
	return "backend"
}

func localMode(cfg *config.Config, fromFallback bool) string {
	if fromFallback {
		return "local_fallback"
	}
	return modeLabel(cfg)
}

// commonElevAudit builds baseline elevation audit fields (AGENT_RULES §6: SID, path, hash, device id, working dir; argv never logged raw).
func (s *Server) commonElevAudit(req Request, exePath, wd, fileSHA string) audit.Entry {
	e := audit.Entry{
		Event:              "elevation",
		UserSID:            req.UserSID,
		ExePath:            exePath,
		SHA256:             fileSHA,
		WorkingDirectory:   wd,
		ArgumentsPresent:   strings.TrimSpace(req.Args) != "",
	}
	if s.Backend != nil {
		if id := strings.TrimSpace(s.Backend.DeviceID()); id != "" {
			e.DeviceID = id
		}
	}
	return e
}

func (s *Server) writeAudit(e audit.Entry) {
	if s.Log == nil {
		return
	}
	s.Log.Write(e)
}

func writeResponse(conn net.Conn, resp Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = conn.Write(b)
	return err
}
