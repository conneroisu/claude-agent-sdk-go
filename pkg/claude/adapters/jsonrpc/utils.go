package jsonrpc

import (
	"crypto/rand"
	"encoding/hex"
)

// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read error is documented as always nil

	return hex.EncodeToString(b)
}
