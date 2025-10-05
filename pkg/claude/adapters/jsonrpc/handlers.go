package jsonrpc

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

func (a *Adapter) handleControlResponse(msg map[string]any) {
	response, ok := msg["response"].(map[string]any)
	if !ok {
		return
	}

	requestID, ok := response["request_id"].(string)
	if !ok {
		return
	}

	a.mu.Lock()
	ch, exists := a.pendingReqs[requestID]
	delete(a.pendingReqs, requestID)
	a.mu.Unlock()

	if !exists {
		return
	}

	select {
	case ch <- result{data: response}:
	default:
	}
}

// HandleControlRequest implements ports.ProtocolHandler.
func (*Adapter) HandleControlRequest(
	_ context.Context,
	_ map[string]any,
	_ ports.ControlDependencies,
) (map[string]any, error) {
	// Implementation would handle hook_callback, can_use_tool, etc.
	// Simplified for now - full implementation needed
	return map[string]any{"status": "ok"}, nil
}

func (a *Adapter) processControlRequest(
	ctx context.Context,
	msg map[string]any,
	deps ports.ControlDependencies,
) {
	// Delegate to public method
	_, _ = a.HandleControlRequest(ctx, msg, deps)
}
