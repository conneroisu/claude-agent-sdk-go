package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

// sendControlResponse sends a control response back to the CLI.
func (q *queryImpl) sendControlResponse(
	ctx context.Context,
	requestID string,
	responseData map[string]any,
	err error,
) error {
	var response SDKControlResponse
	response.BaseMessage = BaseMessage{
		UUIDField:      uuid.New(),
		SessionIDField: q.sessionID,
	}

	if err != nil {
		// Send error response
		response.Response = ControlErrorResponse{
			SubtypeField:   "error",
			RequestIDField: requestID,
			Error:          err.Error(),
		}
	} else {
		// Send success response
		jsonValueMap := make(map[string]JSONValue)
		for k, v := range responseData {
			jsonBytes, marshalErr := json.Marshal(v)
			if marshalErr != nil {
				return clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed,
					fmt.Sprintf("failed to marshal response data for key %s", k),
					marshalErr).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			jsonValueMap[k] = jsonBytes
		}

		response.Response = ControlSuccessResponse{
			SubtypeField:   "success",
			RequestIDField: requestID,
			Response:       jsonValueMap,
		}
	}

	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal control response",
			marshalErr,
		).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_response")
	}

	return q.proc.Transport().Write(ctx, data)
}

// sendControlRequest sends a control request and waits for response.
func (q *queryImpl) sendControlRequest(
	ctx context.Context,
	request ControlRequestVariant,
) (map[string]any, error) {
	// Generate unique request ID
	q.mu.Lock()
	q.requestCounter++
	counter := q.requestCounter
	q.mu.Unlock()

	requestID := fmt.Sprintf(requestIDFormat, counter, uuid.New().String()[:8])

	// Create channel for response
	respChan := make(chan *SDKControlResponse, 1)
	q.mu.Lock()
	q.pendingControlResponses[requestID] = respChan
	q.mu.Unlock()

	// Build and send request
	controlReq := SDKControlRequest{
		BaseMessage: BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: q.sessionID,
		},
		RequestID: requestID,
		Request:   request,
	}

	data, err := json.Marshal(controlReq)
	if err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal control request",
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

		return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
			"failed to send control request", err).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_request")
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		// Check response type
		switch r := resp.Response.(type) {
		case ControlSuccessResponse:
			// Convert JSONValue map to any map
			result := make(map[string]any)
			for k, v := range r.Response {
				result[k] = v
			}

			return result, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("control request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("unexpected control response type: %T", r), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		}
	case <-ctx.Done():
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return nil, ctx.Err()
	}
}
