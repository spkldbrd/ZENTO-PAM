package backend

import "time"

// RegisterOS is used by device_windows / device_stub (not sent in Phase 1 register body).
type RegisterOS struct {
	Edition string `json:"edition"`
	Version string `json:"version"`
	Build   string `json:"build"`
}

// --- POST /agent/register (API_SPEC.md) ---

type RegisterRequest struct {
	Hostname     string `json:"hostname"`
	AgentVersion string `json:"agent_version"`
}

type RegisterResponse struct {
	DeviceID string `json:"deviceId"`
}

// --- POST /agent/elevation-request ---

type ElevationRequestPayload struct {
	DeviceID  string `json:"device_id"`
	User      string `json:"user"`
	ExePath   string `json:"exe_path"`
	Hash      string `json:"hash"`
	Publisher string `json:"publisher"`
}

// ElevationPostResponse is returned from POST (200 OK).
type ElevationPostResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ElevationPollResponse is returned from GET /agent/elevation-requests/:id
type ElevationPollResponse struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

// Grant is reserved for future APIs; Phase 1 poll responses omit it.
type Grant struct {
	Type        string           `json:"type"`
	Constraints GrantConstraints `json:"constraints"`
}

type GrantConstraints struct {
	AllowedPath   string `json:"allowed_path"`
	AllowedSHA256 string `json:"allowed_sha256"`
}
