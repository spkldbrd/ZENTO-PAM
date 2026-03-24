package backend

import (
	"crypto/rand"
	"fmt"
)

// NewCorrelationID returns a random UUID v4 string for local audit correlation (Phase 1 API has no idempotency header).
func NewCorrelationID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint32(b[4])<<8|uint32(b[5]),
		uint32(b[6])<<8|uint32(b[7]),
		uint32(b[8])<<8|uint32(b[9]),
		b[10:16])
}
