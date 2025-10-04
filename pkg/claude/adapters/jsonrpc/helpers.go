package jsonrpc

import (
	"crypto/rand"
	"encoding/hex"
)

// randomHex generates a random hex string of n bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// getStringPtr extracts an optional string pointer from a map.
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}
	return nil
}
