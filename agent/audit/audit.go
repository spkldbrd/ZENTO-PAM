package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is one JSON line in logs/agent.log.
type Entry struct {
	TimestampUTC string `json:"timestamp_utc"`
	Event        string `json:"event,omitempty"` // elevation | registration
	UserSID      string `json:"user_sid,omitempty"`
	ExePath      string `json:"exe_path,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Decision     string `json:"decision,omitempty"` // allowed | denied
	Reason       string `json:"reason,omitempty"`
	Result       string `json:"result,omitempty"` // launched | skipped | error message
	// Enrolled device (backend mode); also set on registration when applicable
	DeviceID string `json:"device_id,omitempty"`
	// Effective working directory for launch (never log raw argv; see arguments_present)
	WorkingDirectory string `json:"working_directory,omitempty"`
	// True when the client sent non-empty arguments (content is not logged)
	ArgumentsPresent bool `json:"arguments_present,omitempty"`
	// SHA-256 (hex, lowercase) of the raw args string from the pipe (empty string if args empty)
	ArgsSHA256 string `json:"args_sha256,omitempty"`
	// Backend integration (elevation flow)
	Mode               string `json:"mode,omitempty"` // backend | local_fallback | local_only
	BackendRequestID   string `json:"backend_request_id,omitempty"`
	BackendStatus      string `json:"backend_status,omitempty"`
	CorrelationID      string `json:"correlation_id,omitempty"`
}

// Logger writes mutex-protected JSON lines to a file.
type Logger struct {
	mu  sync.Mutex
	f   *os.File
	err error
}

// New opens or creates the log file under baseDir/logs/agent.log.
func New(baseDir string) (*Logger, error) {
	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(logDir, "agent.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// Write appends one JSON line.
func (l *Logger) Write(e Entry) {
	if l == nil || l.f == nil {
		return
	}
	if e.TimestampUTC == "" {
		e.TimestampUTC = time.Now().UTC().Format(time.RFC3339Nano)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = l.f.Write(append(b, '\n'))
	_ = l.f.Sync()
}
