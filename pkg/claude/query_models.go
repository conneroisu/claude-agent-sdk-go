package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

// SupportedModels returns available models.
func (q *queryImpl) SupportedModels(ctx context.Context) ([]ModelInfo, error) {
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
		"request": map[string]any{
			"subtype": "supportedModels",
		},
	}

	data, err := json.Marshal(controlReq)
	if err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal SupportedModels request",
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

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			"failed to send SupportedModels request",
			err,
		).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_request")
	}

	select {
	case resp := <-respChan:
		switch r := resp.Response.(type) {
		case ControlSuccessResponse:
			modelsData, ok := r.Response["models"]
			if !ok {
				return make([]ModelInfo, 0), nil
			}
			data, err := json.Marshal(modelsData)
			if err != nil {
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed,
					"failed to marshal models data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			var models []ModelInfo
			if err := json.Unmarshal(data, &models); err != nil {
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed,
					"failed to parse models data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}

			return models, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("SupportedModels request failed: %s", r.Error), nil).
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

// McpServerStatus returns MCP server status.
func (q *queryImpl) McpServerStatus(ctx context.Context) ([]McpServerStatus, error) {
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
		"request": map[string]any{
			"subtype": "mcpServerStatus",
		},
	}

	data, err := json.Marshal(controlReq)
	if err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal McpServerStatus request",
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

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			"failed to send McpServerStatus request",
			err,
		).
			WithSessionID(q.sessionID).
			WithRequestID(requestID).
			WithMessageType("control_request")
	}

	select {
	case resp := <-respChan:
		switch r := resp.Response.(type) {
		case ControlSuccessResponse:
			serversData, ok := r.Response["servers"]
			if !ok {
				return make([]McpServerStatus, 0), nil
			}
			data, err := json.Marshal(serversData)
			if err != nil {
				return nil, clauderrs.NewProtocolError(
					clauderrs.ErrCodeMessageParseFailed,
					"failed to marshal servers data",
					err,
				).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			var servers []McpServerStatus
			if err := json.Unmarshal(data, &servers); err != nil {
				return nil, clauderrs.NewProtocolError(
					clauderrs.ErrCodeMessageParseFailed,
					"failed to parse servers data",
					err,
				).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}

			return servers, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeProtocolError,
				fmt.Sprintf("McpServerStatus request failed: %s", r.Error),
				nil,
			).
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
