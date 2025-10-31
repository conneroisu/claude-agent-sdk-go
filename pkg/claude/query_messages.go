package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

// readMessages reads messages from the process.
func (q *queryImpl) readMessages() {
	defer close(q.msgChan)

	for {
		select {
		case <-q.closeChan:
			return
		default:
			msg, err := q.readMessage()
			if err != nil {
				q.handleReadError(err)
				return
			}

			if msg != nil {
				q.msgChan <- msg
			}
		}
	}
}

// handleReadError handles errors during message reading.
func (q *queryImpl) handleReadError(err error) {
	if err == io.EOF {
		return
	}
	q.errChan <- err
}

// readMessage reads a single message from the process.
func (q *queryImpl) readMessage() (SDKMessage, error) {
	data, err := q.proc.Transport().Read(context.Background())
	if err != nil {
		return nil, err
	}

	// Parse the message type first
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse message envelope",
			err,
		).
			WithSessionID(q.sessionID)
	}

	// Handle control responses
	if envelope.Type == messageTypeControlResponse {
		var resp SDKControlResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse control response",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("control_response")
		}

		// Route to the pending request
		q.mu.Lock()
		if ch, ok := q.pendingControlResponses[resp.Response.RequestID()]; ok {
			ch <- &resp
			delete(q.pendingControlResponses, resp.Response.RequestID())
		}
		q.mu.Unlock()

		return nil, nil // Control responses don't go to the message stream
	}

	// Handle incoming control requests from CLI (bidirectional control protocol)
	if envelope.Type == messageTypeControlRequest {
		// Route to control request handler instead of message stream
		select {
		case q.controlRequestChan <- data:
		case <-q.closeChan:
			return nil, io.EOF
		}

		return nil, nil // Control requests don't go to the message stream
	}

	// Decode based on type
	switch envelope.Type {
	case "user":
		var msg SDKUserMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse user message",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("user")
		}

		return &msg, nil

	case "assistant":
		var msg SDKAssistantMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse assistant message",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("assistant")
		}

		return &msg, nil

	case "stream_event":
		var msg SDKStreamEvent
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse stream event",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("stream_event")
		}

		return &msg, nil

	case "system":
		var msg SDKSystemMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse system message",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("system")
		}

		return &msg, nil

	case "result":
		var msg SDKResultMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse result message",
				err,
			).
				WithSessionID(q.sessionID).
				WithMessageType("result")
		}

		return &msg, nil

	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf("unknown message type: %s", envelope.Type),
			nil,
		).
			WithSessionID(q.sessionID).
			WithMessageType(envelope.Type)
	}
}

// SendUserMessage sends a text user message to the process.
func (q *queryImpl) SendUserMessage(ctx context.Context, text string) error {
	return q.SendUserMessageWithContent(ctx, []ContentBlock{
		TextContentBlock{
			Type: "text",
			Text: text,
		},
	})
}

// SendUserMessageWithContent sends a user message with structured
// content blocks.
func (q *queryImpl) SendUserMessageWithContent(
	ctx context.Context,
	content []ContentBlock,
) error {
	msg := SDKUserMessage{
		BaseMessage: BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: q.sessionID,
		},
		TypeField: "user",
		Message: APIUserMessage{
			Role:    "user",
			Content: content,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal user message", err).
			WithSessionID(q.sessionID).
			WithMessageType("user")
	}

	return q.proc.Transport().Write(ctx, data)
}

// Next returns the next message from the query.
func (q *queryImpl) Next(ctx context.Context) (SDKMessage, error) {
	select {
	case msg, ok := <-q.msgChan:
		if !ok {
			return nil, io.EOF
		}

		return msg, nil
	case err := <-q.errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-q.closeChan:
		return nil, io.EOF
	}
}

// Close closes the query and cleans up resources.
func (q *queryImpl) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.closeChan)

	if q.proc != nil {
		return q.proc.Close()
	}

	return nil
}

// QueryFunc creates a new query session.
func QueryFunc(prompt string, opts *Options) (Query, error) {
	return newQueryImpl(prompt, opts)
}
