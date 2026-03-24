package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"pam-platform/agent/config"
)

// Client calls the PAM backend Phase 1 API (API_SPEC.md).
type Client struct {
	cfg     *config.Config
	baseDir string
	http    *http.Client
	mu      sync.Mutex
	state   AgentState
}

// FinalOutcome is the terminal result after POST + polling.
type FinalOutcome struct {
	RequestID string
	Status    string // ALLOWED, DENIED, TIMEOUT (uppercase for ipc/server.go)
	Grant     *Grant
}

// Connect registers the device on startup and returns a client, or (nil, nil) if backend is disabled.
func Connect(ctx context.Context, cfg *config.Config, baseDir string) (*Client, error) {
	if cfg == nil || !cfg.BackendEnabled() {
		return nil, nil
	}

	c := &Client{
		cfg:     cfg,
		baseDir: baseDir,
		http: &http.Client{
			Timeout: cfg.HTTPTimeout(),
		},
	}

	if err := c.Register(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// Register performs POST /agent/register and persists device_id.
func (c *Client) Register(ctx context.Context) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	body := RegisterRequest{
		Hostname:     host,
		AgentVersion: c.cfg.AgentVersion,
	}

	u, err := url.JoinPath(c.cfg.BackendBaseURL, "agent", "register")
	if err != nil {
		return err
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register: http %d: %s", resp.StatusCode, trimErr(rb))
	}

	var out RegisterResponse
	if err := json.Unmarshal(rb, &out); err != nil {
		return fmt.Errorf("register decode: %w", err)
	}
	if out.DeviceID == "" {
		return fmt.Errorf("register: missing deviceId")
	}

	c.mu.Lock()
	c.state = AgentState{DeviceID: out.DeviceID}
	c.mu.Unlock()

	if err := SaveState(c.baseDir, &c.state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	return nil
}

// DeviceID returns the enrolled device id after successful registration.
func (c *Client) DeviceID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.DeviceID
}

// SubmitAndWait POSTs /agent/elevation-request and polls GET /agent/elevation-requests/:id until terminal or timeout.
func (c *Client) SubmitAndWait(ctx context.Context, payload *ElevationRequestPayload) (FinalOutcome, error) {
	if payload == nil {
		return FinalOutcome{}, fmt.Errorf("nil elevation payload")
	}
	c.mu.Lock()
	if c.state.DeviceID == "" {
		c.mu.Unlock()
		return FinalOutcome{}, fmt.Errorf("device not registered")
	}
	payload.DeviceID = c.state.DeviceID
	c.mu.Unlock()

	postURL, err := url.JoinPath(c.cfg.BackendBaseURL, "agent", "elevation-request")
	if err != nil {
		return FinalOutcome{}, err
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return FinalOutcome{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(buf))
	if err != nil {
		return FinalOutcome{}, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return FinalOutcome{}, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return FinalOutcome{}, fmt.Errorf("elevation-request: http %d: %s", resp.StatusCode, trimErr(rb))
	}

	var postOut ElevationPostResponse
	if err := json.Unmarshal(rb, &postOut); err != nil {
		return FinalOutcome{}, fmt.Errorf("decode elevation response: %w", err)
	}
	if postOut.ID == "" {
		return FinalOutcome{}, fmt.Errorf("elevation-request: missing id")
	}

	switch strings.ToLower(strings.TrimSpace(postOut.Status)) {
	case "approved":
		return FinalOutcome{RequestID: postOut.ID, Status: "ALLOWED", Grant: nil}, nil
	case "denied":
		return FinalOutcome{RequestID: postOut.ID, Status: "DENIED"}, nil
	case "pending":
		return c.pollUntilTerminal(ctx, postOut.ID)
	default:
		return FinalOutcome{}, fmt.Errorf("unexpected initial status %q", postOut.Status)
	}
}

func (c *Client) pollUntilTerminal(ctx context.Context, requestID string) (FinalOutcome, error) {
	deadline := c.cfg.ApprovalDeadline()
	wait := c.cfg.PollingInterval()

	for {
		if !time.Now().Before(deadline) {
			return FinalOutcome{RequestID: requestID, Status: "TIMEOUT"}, fmt.Errorf("approval timeout")
		}
		select {
		case <-ctx.Done():
			return FinalOutcome{}, ctx.Err()
		default:
		}

		rem := time.Until(deadline)
		if wait > rem {
			wait = rem
		}
		if wait > 0 {
			t := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				t.Stop()
				return FinalOutcome{}, ctx.Err()
			case <-t.C:
			}
		}
		wait = c.cfg.PollingInterval()

		st, err := c.getRequest(ctx, requestID)
		if err != nil {
			return FinalOutcome{}, err
		}

		switch strings.ToLower(strings.TrimSpace(st.Status)) {
		case "approved":
			return FinalOutcome{RequestID: st.ID, Status: "ALLOWED", Grant: nil}, nil
		case "denied":
			return FinalOutcome{RequestID: st.ID, Status: "DENIED"}, nil
		case "pending":
			continue
		default:
			return FinalOutcome{}, fmt.Errorf("unknown status %q", st.Status)
		}
	}
}

func (c *Client) getRequest(ctx context.Context, requestID string) (ElevationPollResponse, error) {
	u, err := url.JoinPath(c.cfg.BackendBaseURL, "agent", "elevation-requests", requestID)
	if err != nil {
		return ElevationPollResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ElevationPollResponse{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return ElevationPollResponse{}, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ElevationPollResponse{}, fmt.Errorf("get elevation-request: http %d: %s", resp.StatusCode, trimErr(rb))
	}
	var st ElevationPollResponse
	if err := json.Unmarshal(rb, &st); err != nil {
		return ElevationPollResponse{}, err
	}
	return st, nil
}

func trimErr(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
}
