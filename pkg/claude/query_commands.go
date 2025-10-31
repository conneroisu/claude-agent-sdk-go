package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

// SupportedCommands returns available slash commands.
func (q *queryImpl) SupportedCommands(
	ctx context.Context,
) ([]SlashCommand, error) {
	// Use generic approach for control requests without specific types
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
			"subtype": "supportedCommands",
		},
	}

	data, err := json.Marshal(controlReq)
	if err != nil {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()

		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal SupportedCommands request",
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
			"failed to send SupportedCommands request",
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
			commandsData, ok := r.Response["commands"]
			if !ok {
				return make([]SlashCommand, 0), nil
			}
			data, err := json.Marshal(commandsData)
			if err != nil {
				return nil,
					clauderrs.NewProtocolError(
						clauderrs.ErrCodeMessageParseFailed,
						"failed to marshal commands data", err).
						WithSessionID(q.sessionID).
						WithRequestID(requestID).
						WithMessageType("control_response")
			}
			var commands []SlashCommand
			err = json.Unmarshal(data, &commands)
			if err != nil {
				return nil,
					clauderrs.NewProtocolError(
						clauderrs.ErrCodeMessageParseFailed,
						"failed to parse commands data", err).
						WithSessionID(q.sessionID).
						WithRequestID(requestID).
						WithMessageType("control_response")
			}

			return commands, nil
		case ControlErrorResponse:
			return nil,
				clauderrs.NewProtocolError(
					clauderrs.ErrCodeProtocolError,
					fmt.Sprintf("SupportedCommands request failed: %s", r.Error), nil).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
		default:
			return nil,
				clauderrs.NewProtocolError(
					clauderrs.ErrCodeProtocolError,
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
