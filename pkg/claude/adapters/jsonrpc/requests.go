package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	// defaultRequestTimeout is the default timeout for control requests.
	defaultRequestTimeout = 60 * time.Second
)

// Initialize implements ports.ProtocolHandler.
// It sends an initialization control request to configure the protocol session.
func (a *Adapter) Initialize(
	ctx context.Context,
	config map[string]any,
) (map[string]any, error) {
	// Build initialization request with config params
	req := map[string]any{
		"type":   "control_request",
		"method": "initialize",
		"params": config,
	}

	return a.SendControlRequest(ctx, req)
}

// SendControlRequest sends a control request and waits for response.
// It handles request ID generation, tracking, timeout, and cleanup.
// Returns the response data or an error if the request fails or times out.
func (a *Adapter) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	// Generate unique request ID for response correlation
	requestID := a.generateRequestID()
	resCh := make(chan result, 1)

	// Register pending request channel
	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()

	// Cleanup pending request when done
	defer func() {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
	}()

	// Add request ID to message
	req["request_id"] = requestID

	// Serialize request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Send request over transport with newline delimiter
	if err := a.transport.Write(ctx, string(data)+"\n"); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	select {
	case res := <-resCh:
		return res.data, res.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("request timeout: %w", timeoutCtx.Err())
	}
}

// generateRequestID creates a unique request identifier.
// It combines an incrementing counter with timestamp for uniqueness.
func (a *Adapter) generateRequestID() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCounter++

	return fmt.Sprintf("req_%d_%d", a.requestCounter, time.Now().Unix())
}
