package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const requestIDHexLength = 4

// Initialize is a no-op - initialization happens implicitly in StartMessageRouter.
func (*Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
	return nil, nil
}

// SendControlRequest sends a control request and waits for response.
// This method handles all request ID generation and timeout logic.
func (a *Adapter) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	// Generate unique request ID
	a.mu.Lock()
	a.requestCounter++
	requestID := fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(requestIDHexLength))
	a.mu.Unlock()

	// Create result channel for this request
	resCh := make(chan result, 1)
	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()

	// Build control request envelope
	controlReq := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    req,
	}

	// Send via transport
	reqBytes, err := json.Marshal(controlReq)
	if err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		return nil, fmt.Errorf("marshal control request: %w", err)
	}

	if err := a.transport.Write(ctx, string(reqBytes)+"\n"); err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		return nil, fmt.Errorf("write control request: %w", err)
	}

	// Wait for response with 60s timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("control request timeout: %s", req["subtype"])
		}
		return nil, timeoutCtx.Err()
	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}
		return res.data, nil
	}
}
