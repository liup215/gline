package memory

import (
	"crypto/rand"
	"encoding/hex"
)

// genID generates a short random hex ID (slim, no UUID dep).
func genID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Truncate limits s to maxRunes runes, appending "…" if truncated.
func Truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}
