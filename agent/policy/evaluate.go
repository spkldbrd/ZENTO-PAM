package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// Evaluate checks exePath against the policy using file hash and Authenticode publisher (Windows).
func (p *Policy) Evaluate(exePath string) Result {
	if p == nil {
		return Result{Allowed: false, Reason: "policy not loaded"}
	}
	if len(p.AllowedPublishers) == 0 && len(p.AllowedHashes) == 0 {
		return Result{Allowed: false, Reason: "deny all: empty allowed_publishers and allowed_hashes"}
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		return Result{Allowed: false, Reason: "read exe: " + err.Error()}
	}
	sum := sha256.Sum256(data)
	hashHex := strings.ToLower(hex.EncodeToString(sum[:]))

	hashAllowed := false
	if len(p.AllowedHashes) > 0 {
		for _, h := range p.AllowedHashes {
			if h != "" && h == hashHex {
				hashAllowed = true
				break
			}
		}
	}

	// Hash-only policy: never resolve publisher.
	if len(p.AllowedHashes) > 0 && len(p.AllowedPublishers) == 0 {
		if !hashAllowed {
			return Result{Allowed: false, Reason: "hash not in allowed_hashes", HashHex: hashHex}
		}
		return Result{Allowed: true, Reason: "matched allowed_hashes", HashHex: hashHex}
	}

	// OR mode: hash match wins without publisher lookup.
	if len(p.AllowedHashes) > 0 && len(p.AllowedPublishers) > 0 && hashAllowed {
		return Result{Allowed: true, Reason: "matched allowed_hashes", HashHex: hashHex}
	}

	// Publisher required (publisher-only, or OR mode with hash mismatch).
	var pub string
	var pubErr error
	pub, pubErr = filePublisher(exePath)
	if pubErr != nil {
		pub = ""
	}

	pubAllowed := false
	if pub != "" {
		for _, want := range p.AllowedPublishers {
			if want == "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(pub)) {
				pubAllowed = true
				break
			}
		}
	}

	var allowed bool
	var reason string
	if len(p.AllowedHashes) > 0 && len(p.AllowedPublishers) > 0 {
		allowed = hashAllowed || pubAllowed
		if !allowed {
			reason = "not in allowed_hashes and publisher not in allowed_publishers"
		} else if hashAllowed {
			reason = "matched allowed_hashes"
		} else {
			reason = "matched allowed_publishers"
		}
	} else {
		allowed = pubAllowed
		if !allowed {
			if pub == "" {
				if pubErr != nil {
					reason = "could not read publisher: " + pubErr.Error()
				} else {
					reason = "no publisher on binary"
				}
			} else {
				reason = "publisher not in allowed_publishers"
			}
		} else {
			reason = "matched allowed_publishers"
		}
	}

	return Result{
		Allowed:   allowed,
		Reason:    reason,
		Publisher: pub,
		HashHex:   hashHex,
	}
}
