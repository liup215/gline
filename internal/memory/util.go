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
