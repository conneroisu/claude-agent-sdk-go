package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

// Interrupt interrupts the current query.
func (q *queryImpl) Interrupt(ctx context.Context) error {
	_, err := q.sendControlRequest(ctx, SDKControlInterruptRequest{})

	return err
}

// SetPermissionMode changes the permission mode.
func (q *queryImpl) SetPermissionMode(
	ctx context.Context,
	mode PermissionMode,
) error {
	_, err := q.sendControlRequest(ctx, SDKControlSetPermissionModeRequest{
		Mode: string(mode),
	})

	return err
}

// SetModel changes the model.
func (q *queryImpl) SetModel(ctx context.Context, model *string) error {
	// Create a request with the model field
	// Note: We need to add this request type to messages.go
	request := map[string]any{
		"subtype": "setModel",
		"model":   model,
	}

	// For now, use a generic approach since we don't have
	// SDKControlSetModelRequest
	q.mu.Lock()
	q.requestCounter++
	counter := q.requestCounter
	q.mu.Unlock()

	requestID := fmt.Sprintf(requestIDFormat, counter, uuid.New().String()[:8])

	respChan := make(chan *SDKControlResponse, 1)
	q.mu.Lock()
	q.pendingControlResponses[requestID] = respChan
	q.mu.Unlock()

	controlReq := map[string]any{
		fieldType:      messageTypeControlRequest,
		fieldUUID:      uuid.New().String(),
		fieldSessionID: q.sessionID,
		fieldRequestID: requestID,
		fieldRequest:   request,
	}

	data, err := json.Marshal(controlReq)
	if err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal SetModel request",
			err,
		).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_request")
	}

	if err := q.proc.Transport().Write(ctx, data); err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
			"failed to send SetModel request", err).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_request")
	}

	select {
	case resp := <-respChan:
		switch r := resp.Response.(type) {
		case ControlSuccessResponse:
			return nil
		case ControlErrorResponse:
			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("SetModel request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("unexpected control response type: %T", r), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		}
	case <-ctx.Done():
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return ctx.Err()
	}
}
