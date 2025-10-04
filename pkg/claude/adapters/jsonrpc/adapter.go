package jsonrpc

import (
	"context"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol.
// This is an INFRASTRUCTURE adapter - it handles protocol state management.
// The domain services delegate protocol concerns to this adapter.
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
// The adapter wraps a transport and manages control protocol state.
func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

// permissionServiceAdapter wraps a ports.PermissionService to provide the CheckToolUse method
type permissionServiceAdapter struct {
	svc ports.PermissionService
}

func (p *permissionServiceAdapter) CanUseTool(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	return p.svc.CanUseTool(ctx, req)
}
