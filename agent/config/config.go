package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Config is loaded from config.json next to the agent binary (optional).
// If the file is missing or backend_base_url is empty, the agent runs in local-only mode.
type Config struct {
	BackendBaseURL        string `json:"backend_base_url"`
	// EnrollmentKey is reserved for future authenticated APIs; Phase 1 backend ignores it.
	EnrollmentKey         string `json:"enrollment_key"`
	PollingIntervalMS     int    `json:"polling_interval_ms"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
	LocalFallback         bool   `json:"local_fallback"`
	AgentVersion          string `json:"agent_version"`
}

// Default returns defaults when fields are zero.
func (c *Config) Default() {
	if c.PollingIntervalMS <= 0 {
		c.PollingIntervalMS = 2000
	}
	if c.RequestTimeoutSeconds <= 0 {
		c.RequestTimeoutSeconds = 180
	}
	if c.AgentVersion == "" {
		c.AgentVersion = "0.2.0"
	}
}

// Load reads JSON from path. Missing file returns empty Config (local-only).
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c := &Config{}
			c.Default()
			return c, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.BackendBaseURL = strings.TrimSpace(strings.TrimSuffix(c.BackendBaseURL, "/"))
	c.Default()
	return &c, nil
}

// BackendEnabled is true when a base URL is configured (registration may still fail).
func (c *Config) BackendEnabled() bool {
	return c != nil && c.BackendBaseURL != ""
}

// PollingInterval returns polling delay between status checks.
func (c *Config) PollingInterval() time.Duration {
	return time.Duration(c.PollingIntervalMS) * time.Millisecond
}

// ApprovalDeadline is the maximum time to wait for PENDING -> terminal after submitting a request.
func (c *Config) ApprovalDeadline() time.Time {
	return time.Now().Add(time.Duration(c.RequestTimeoutSeconds) * time.Second)
}

// HTTPTimeout per outbound HTTP call (bounded by overall approval window).
func (c *Config) HTTPTimeout() time.Duration {
	s := c.RequestTimeoutSeconds
	if s <= 0 {
		s = 180
	}
	d := time.Duration(s) * time.Second / 3
	if d < 10*time.Second {
		return 10 * time.Second
	}
	if d > 60*time.Second {
		return 60 * time.Second
	}
	return d
}
