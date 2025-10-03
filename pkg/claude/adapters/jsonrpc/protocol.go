// Package jsonrpc implements the JSON-RPC control protocol adapter.
//
// This adapter implements the ProtocolHandler port and manages all
// control protocol state (pending requests, request IDs, etc.).
package jsonrpc

import (
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol.
// This is an INFRASTRUCTURE adapter - it handles protocol state.
type Adapter struct {
	transport ports.Transport

	// Control protocol state (managed by adapter, not domain)
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// Verify interface compliance at compile time.
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
	data map[string]any
	err  error
}

// NewAdapter creates a new JSON-RPC protocol adapter.
func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}
