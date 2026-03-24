package backend

import (
	"path/filepath"
	"strings"
)

// GrantAllowsPathAndHash enforces grant.constraints when the backend includes them.
func GrantAllowsPathAndHash(g *Grant, exePath, fileSHA256 string) bool {
	if g == nil {
		return true
	}
	c := g.Constraints
	if c.AllowedPath == "" && c.AllowedSHA256 == "" {
		return true
	}
	if c.AllowedPath != "" {
		if !strings.EqualFold(filepath.Clean(c.AllowedPath), filepath.Clean(exePath)) {
			return false
		}
	}
	if c.AllowedSHA256 != "" {
		if !strings.EqualFold(strings.TrimSpace(c.AllowedSHA256), strings.TrimSpace(fileSHA256)) {
			return false
		}
	}
	return true
}
