package policy

import (
	"encoding/json"
	"os"
	"strings"
)

// Policy is loaded from policy/policy.json next to the service install root.
// Evaluation rules (Phase 1):
//   - If both allowed_publishers and allowed_hashes are empty: deny all.
//   - If allowed_hashes is non-empty: allow if hash matches OR publisher matches allowed_publishers (when non-empty).
//   - If allowed_hashes is empty: allow only if publisher matches one of allowed_publishers (case-insensitive trim).
type Policy struct {
	AllowedPublishers []string `json:"allowed_publishers"`
	AllowedHashes     []string `json:"allowed_hashes"`
}

// Load reads and parses policy from path.
func Load(path string) (*Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Policy
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	for i := range p.AllowedPublishers {
		p.AllowedPublishers[i] = strings.TrimSpace(p.AllowedPublishers[i])
	}
	for i := range p.AllowedHashes {
		p.AllowedHashes[i] = strings.ToLower(strings.TrimSpace(p.AllowedHashes[i]))
	}
	return &p, nil
}

// Result is the outcome of evaluating a file against the policy.
type Result struct {
	Allowed   bool
	Reason    string
	Publisher string
	HashHex   string
}
