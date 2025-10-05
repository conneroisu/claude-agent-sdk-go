// Package jsonrpc provides JSON-RPC control protocol handling.
package jsonrpc

import (
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for JSON-RPC control protocol.
type Adapter struct {
	transport      ports.Transport
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// NewAdapter creates a new JSON-RPC protocol adapter.
func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

type result struct {
	data map[string]any
	err  error
}

// Compile-time interface verification.
var _ ports.ProtocolHandler = (*Adapter)(nil)
