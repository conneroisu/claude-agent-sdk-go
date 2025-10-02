// Package jsonrpc implements the JSON-RPC protocol adapter for the Claude SDK.
// It handles control protocol state management and message routing between
// the transport layer and the SDK core.
package jsonrpc

import (
	"context"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol
// This is an INFRASTRUCTURE adapter - it handles protocol state management
type Adapter struct {
	transport ports.Transport
	// Control protocol state (managed by adapter, not domain)
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// Verify interface compliance at compile time
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
	data map[string]any
	err  error
}

// routerChannels holds the channels used by the message router
type routerChannels struct {
	transportMsg <-chan map[string]any
	transportErr <-chan error
	msg          chan<- map[string]any
	err          chan<- error
}

// NewAdapter creates a new JSON-RPC protocol adapter
func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

// Initialize is a no-op - initialization happens implicitly in
// StartMessageRouter
func (*Adapter) Initialize(
	_ context.Context,
	_ map[string]any,
) (map[string]any, error) {
	return nil, nil
}
