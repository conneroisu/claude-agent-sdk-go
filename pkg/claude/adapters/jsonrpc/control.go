package jsonrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Initialize is a no-op - initialization happens implicitly.
//nolint:revive,staticcheck // receiver-naming: unused receiver for interface
//nolint:revive // receiver-naming,unused-parameter: Interface implementation
func (*Adapter) Initialize(
	_ context.Context,
	_ map[string]any,
) (map[string]any, error) {
	return nil, nil
}

// SendControlRequest sends a control request and waits for
// response. This method handles all request ID generation and
// timeout logic.
func (a *Adapter) SendControlRequest(
	ctx context.Context,
	req map[string]any,
) (map[string]any, error) {
	requestID := a.generateRequestID()
	resCh := make(chan result, 1)

	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()

	controlReq := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    req,
	}

	reqBytes, err := json.Marshal(controlReq)
	if err != nil {
		a.cleanupRequest(requestID)

		return nil, fmt.Errorf(
			"marshal control request: %w",
			err,
		)
	}

	if err := a.transport.Write(
		ctx,
		string(reqBytes)+"\n",
	); err != nil {
		a.cleanupRequest(requestID)

		return nil, fmt.Errorf(
			"write control request: %w",
			err,
		)
	}

	return a.waitForResponse(ctx, requestID, resCh, req)
}

// generateRequestID creates a unique request ID.
func (a *Adapter) generateRequestID() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCounter++

	return fmt.Sprintf(
		"req_%d_%s",
		a.requestCounter,
		randomHex(4), //nolint:revive // add-constant: magic number for random hex
	)
}

// cleanupRequest removes a pending request.
func (a *Adapter) cleanupRequest(requestID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.pendingReqs, requestID)
}

// waitForResponse waits for a control response with timeout.
func (a *Adapter) waitForResponse(
	ctx context.Context,
	requestID string,
	resCh chan result,
	req map[string]any,
) (map[string]any, error) {
	timeoutCtx, cancel := context.WithTimeout(
		ctx,
		60*time.Second,
	)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		a.cleanupRequest(requestID)
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf(
				"control request timeout: %s",
				req["subtype"],
			)
		}

		return nil, timeoutCtx.Err()

	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}

		return res.data, nil
	}
}

// randomHex generates a random hex string of n bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
