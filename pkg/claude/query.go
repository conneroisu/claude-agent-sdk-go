package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/conneroisu/claude-agent-sdk-go/internal/transport"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

const (
	// Message channel and control request buffer sizes.
	msgChanBufferSize        = 100
	controlRequestChanBuffer = 10

	// Control protocol message types and subtypes.
	messageTypeUser            = "user"
	messageTypeControlResponse = "control_response"
	messageTypeHookCallback    = "hook_callback"

	// Request ID format.
	requestIDFormat = "req_%d_%s"

	// JSON field names.
	fieldType      = "type"
	fieldUUID      = "uuid"
	fieldSessionID = "session_id"
	fieldRequestID = "request_id"
	fieldRequest   = "request"
	fieldSubtype   = "subtype"
)

// Query represents an active query session.
//
// Design Note: The TypeScript SDK's Query interface extends AsyncGenerator,
// allowing for iteration and control methods to be called during iteration.
// In Go, we achieve similar functionality through the Query interface methods,
// though the pattern differs from TypeScript's async generator approach.
// The Next() method provides sequential message access similar to TypeScript's
// for-await-of loop, while control methods (Interrupt, SetModel, etc.) can be
// called at any time during iteration.
//
// Key differences from TypeScript:
// - TypeScript: Uses AsyncGenerator with yield for messages
// - Go: Uses explicit Next() method with error return for idiomatic error handling
// - TypeScript: Control methods can be called on the generator object during iteration
// - Go: Same capability via Query interface methods, but more explicit
//
// Both approaches provide equivalent functionality with idiomatic patterns for each language.
type Query interface {
	// Next returns the next message from the query stream.
	Next(ctx context.Context) (SDKMessage, error)
	// Close closes the query and cleans up resources.
	Close() error

	// SendUserMessage sends a text user message to the process.
	SendUserMessage(ctx context.Context, text string) error
	// SendUserMessageWithContent sends a user message with structured content blocks.
	SendUserMessageWithContent(ctx context.Context, content []ContentBlock) error

	// Interrupt interrupts the current query.
	Interrupt(ctx context.Context) error
	// SetPermissionMode changes the permission mode.
	SetPermissionMode(ctx context.Context, mode PermissionMode) error
	// SetModel changes the model.
	SetModel(ctx context.Context, model *string) error
	// SupportedCommands returns available slash commands.
	SupportedCommands(ctx context.Context) ([]SlashCommand, error)
	// SupportedModels returns available models.
	SupportedModels(ctx context.Context) ([]ModelInfo, error)
	// McpServerStatus returns MCP server status.
	McpServerStatus(ctx context.Context) ([]McpServerStatus, error)
	// GetServerInfo returns the initialization result stored during Initialize.
	GetServerInfo() (map[string]any, error)
}

// queryImpl implements the Query interface.
type queryImpl struct {
	proc                    *transport.Process
	msgChan                 chan SDKMessage
	errChan                 chan error
	closeChan               chan struct{}
	opts                    *Options
	sessionID               string
	mu                      sync.Mutex
	closed                  bool
	requestCounter          int
	pendingControlResponses map[string]chan *SDKControlResponse
	initializationResult    map[string]any
	hookCallbacks           map[string]HookCallback // Maps callback IDs to hook functions
	nextCallbackID          int                     // Counter for generating callback IDs
	controlRequestChan      chan json.RawMessage    // Channel for incoming control requests
}

// newQueryImpl creates a new query implementation.
func newQueryImpl(prompt string, opts *Options) (*queryImpl, error) {
	if opts == nil {
		opts = &Options{}
	}

	q := &queryImpl{
		msgChan:                 make(chan SDKMessage, msgChanBufferSize),
		errChan:                 make(chan error, 1),
		closeChan:               make(chan struct{}),
		opts:                    opts,
		sessionID:               uuid.New().String(),
		pendingControlResponses: make(map[string]chan *SDKControlResponse),
		hookCallbacks:           make(map[string]HookCallback),
		nextCallbackID:          0,
		controlRequestChan:      make(chan json.RawMessage, controlRequestChanBuffer),
	}

	// Start the process
	if err := q.start(prompt); err != nil {
		return nil, err
	}

	return q, nil
}

// start initializes the process and message handling.
func (q *queryImpl) start(prompt string) error {
	// Build process args
	args := q.buildArgs()

	// Build environment
	env := q.buildEnv()

	// Create process config
	config := &transport.ProcessConfig{
		Executable:    q.opts.PathToClaudeCodeExecutable,
		Args:          args,
		Env:           env,
		Cwd:           q.opts.Cwd,
		StderrHandler: q.opts.Stderr,
	}

	// Start process
	proc, err := transport.NewProcess(context.Background(), config)
	if err != nil {
		return clauderrs.CreateProcessError(
			clauderrs.ErrCodeProcessSpawnFailed,
			"failed to start Claude Code process",
			err,
			0,
			"",
		).
			WithCommand(fmt.Sprintf("%s %v", q.opts.PathToClaudeCodeExecutable, args)).
			WithSessionID(q.sessionID)
	}
	q.proc = proc

	// Start message reading goroutine
	go q.readMessages()

	// Start control request handler goroutine
	go q.handleControlRequests()

	// Send initial prompt
	if prompt != "" {
		if err := q.SendUserMessage(context.Background(), prompt); err != nil {
			_ = q.Close()

			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, "failed to send initial prompt", err).
				WithSessionID(q.sessionID).
				WithMessageType("user")
		}
	}

	return nil
}

// buildArgs builds the command line arguments for the process.
func (q *queryImpl) buildArgs() []string {
	// Start with required flags for stream-json protocol
	args := []string{
		"--print",
		"--output-format=stream-json",
		"--input-format=stream-json",
		"--verbose",
	}

	if q.opts.Model != "" {
		args = append(args, "--model", q.opts.Model)
	}

	if q.opts.Continue {
		args = append(args, "--continue")
	}

	if q.opts.Resume != "" {
		args = append(args, "--resume", q.opts.Resume)
	}

	if q.opts.PermissionMode != "" {
		args = append(args, "--permission-mode", string(q.opts.PermissionMode))
	}

	// Add additional directories
	for _, dir := range q.opts.AdditionalDirectories {
		args = append(args, "--add-dir", dir)
	}

	// Add allowed tools
	for _, tool := range q.opts.AllowedTools {
		args = append(args, "--allowed-tools", tool)
	}

	// Add disallowed tools
	for _, tool := range q.opts.DisallowedTools {
		args = append(args, "--disallowed-tools", tool)
	}

	// Add include partial messages flag for streaming
	if q.opts.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}

	return args
}

// buildEnv builds the environment variables for the process.
func (q *queryImpl) buildEnv() []string {
	env := make([]string, 0)

	for key, value := range q.opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

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
	if envelope.Type == ControlRequest {
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

// SendUserMessageWithContent sends a user message with structured content blocks.
func (q *queryImpl) SendUserMessageWithContent(ctx context.Context, content []ContentBlock) error {
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
		return clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to marshal user message", err).
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
	close(q.controlRequestChan)

	if q.proc != nil {
		return q.proc.Close()
	}

	return nil
}

// controlRequestEnvelope represents the envelope for control request messages.
type controlRequestEnvelope struct {
	Request struct {
		Subtype string `json:"subtype"`
	} `json:"request"`
	RequestID string `json:"request_id"`
}

// handleControlRequests processes incoming control requests from the CLI.
func (q *queryImpl) handleControlRequests() {
	for {
		select {
		case <-q.closeChan:
			return
		case data := <-q.controlRequestChan:
			// Parse the control request
			var envelope controlRequestEnvelope
			if err := json.Unmarshal(data, &envelope); err != nil {
				// Can't even parse the request ID, log and continue
				continue
			}

			// Handle the request in the background to avoid blocking
			go q.handleControlRequest(
				context.Background(),
				data,
				envelope.RequestID,
				envelope.Request.Subtype,
			)
		}
	}
}

// handleControlRequest handles a single control request from the CLI.
func (q *queryImpl) handleControlRequest(
	ctx context.Context,
	data json.RawMessage,
	requestID, subtype string,
) {
	var responseData map[string]any
	var err error

	switch subtype {
	case "can_use_tool":
		responseData, err = q.handleCanUseTool(ctx, data)
	case "hook_callback":
		responseData, err = q.handleHookCallback(ctx, data)
	case "mcp_message":
		// TODO: Handle SDK MCP requests when MCP servers are implemented
		err = clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			"mcp_message handling not yet implemented",
			nil,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	default:
		err = clauderrs.NewProtocolError(
			clauderrs.ErrCodeProtocolError,
			fmt.Sprintf("unsupported control request subtype: %s", subtype),
			nil,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Send response back to CLI
	if sendErr := q.sendControlResponse(ctx, requestID, responseData, err); sendErr != nil {
		// Log error but don't fail - the CLI will timeout
		if q.opts.Stderr != nil {
			q.opts.Stderr(fmt.Sprintf("Failed to send control response: %v", sendErr))
		}
	}
}

// handleCanUseTool processes can_use_tool control requests.
func (q *queryImpl) handleCanUseTool(
	ctx context.Context,
	data json.RawMessage,
) (map[string]any, error) {
	var req SDKControlPermissionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse permission request",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Check if canUseTool callback is provided
	if q.opts.CanUseTool == nil {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeCallbackFailed,
			"canUseTool callback is not provided",
			nil,
			"canUseTool",
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert JSONValue map to any map for the callback
	inputMap := make(map[string]JSONValue)
	for k, v := range req.Input {
		inputMap[k] = v
	}

	// Parse permission suggestions
	var suggestions []PermissionUpdate
	// TODO: Parse permission suggestions when needed

	// Call the user's callback
	result, err := q.opts.CanUseTool(ctx, req.ToolName, inputMap, suggestions)
	if err != nil {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeCallbackFailed,
			fmt.Sprintf("canUseTool failed for tool '%s'", req.ToolName),
			err,
			"canUseTool",
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert PermissionResult to response format
	responseData := make(map[string]any)
	switch r := result.(type) {
	case *PermissionAllow:
		responseData["allow"] = true
		if r.UpdatedInput != nil {
			responseData["input"] = r.UpdatedInput
		}
		// TODO: Handle updatedPermissions when control protocol supports it
	case PermissionAllow:
		responseData["allow"] = true
		if r.UpdatedInput != nil {
			responseData["input"] = r.UpdatedInput
		}
	case *PermissionDeny:
		responseData["allow"] = false
		responseData["reason"] = r.Message
		// TODO: Handle interrupt flag when control protocol supports it
	case PermissionDeny:
		responseData["allow"] = false
		responseData["reason"] = r.Message
	default:
		return nil, clauderrs.NewCallbackError(clauderrs.ErrCodeCallbackFailed, fmt.Sprintf("canUseTool invalid return type %T", result), nil, "canUseTool", false).
			WithSessionID(q.sessionID)
	}

	return responseData, nil
}

// handleHookCallback processes hook_callback control requests.
func (q *queryImpl) handleHookCallback(
	ctx context.Context,
	data json.RawMessage,
) (map[string]any, error) {
	var req SDKHookCallbackRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse hook callback request",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("control_request")
	}

	// Look up the callback
	q.mu.Lock()
	callback, ok := q.hookCallbacks[req.CallbackID]
	q.mu.Unlock()

	if !ok {
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeHookFailed,
			fmt.Sprintf("no hook callback found for ID: %s", req.CallbackID),
			nil,
			req.CallbackID,
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Parse the hook input using the decoder
	hookInput, err := DecodeHookInput(req.Input)
	if err != nil {
		return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to parse hook input", err).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	// Call the hook callback
	output, err := callback(ctx, hookInput, req.ToolUseID)
	if err != nil {
		toolUseID := ""
		if req.ToolUseID != nil {
			toolUseID = *req.ToolUseID
		}

		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeHookFailed,
			fmt.Sprintf("hook execution failed for tool use ID: %s", toolUseID),
			err,
			req.CallbackID,
			false,
		).
			WithSessionID(q.sessionID)
	}

	// Convert hook output to response format
	// The hook output should already be in the correct format (JSON-serializable)
	// Marshal and unmarshal to convert to map[string]any
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to marshal hook output",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	var responseData map[string]any
	if err := json.Unmarshal(outputBytes, &responseData); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to unmarshal hook output",
			err,
		).
			WithSessionID(q.sessionID).
			WithMessageType("hook_callback")
	}

	return responseData, nil
}

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
				return clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, fmt.Sprintf("failed to marshal response data for key %s", k), marshalErr).
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

		return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, "failed to send control request", err).
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
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("control request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("unexpected control response type: %T", r), nil).
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

// Interrupt interrupts the current query.
func (q *queryImpl) Interrupt(ctx context.Context) error {
	_, err := q.sendControlRequest(ctx, SDKControlInterruptRequest{})

	return err
}

// SetPermissionMode changes the permission mode.
func (q *queryImpl) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
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

	// For now, use a generic approach since we don't have SDKControlSetModelRequest
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
		fieldType:      ControlRequest,
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

		return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, "failed to send SetModel request", err).
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
			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("SetModel request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("unexpected control response type: %T", r), nil).
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

// SupportedCommands returns available slash commands.
func (q *queryImpl) SupportedCommands(ctx context.Context) ([]SlashCommand, error) {
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
		fieldType:      ControlRequest,
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
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to marshal commands data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			var commands []SlashCommand
			if err := json.Unmarshal(data, &commands); err != nil {
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to parse commands data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}

			return commands, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("SupportedCommands request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("unexpected control response type: %T", r), nil).
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
		fieldType:      ControlRequest,
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
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to marshal models data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			var models []ModelInfo
			if err := json.Unmarshal(data, &models); err != nil {
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to parse models data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}

			return models, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("SupportedModels request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("unexpected control response type: %T", r), nil).
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
		fieldType:      ControlRequest,
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
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to marshal servers data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}
			var servers []McpServerStatus
			if err := json.Unmarshal(data, &servers); err != nil {
				return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeMessageParseFailed, "failed to parse servers data", err).
					WithSessionID(q.sessionID).
					WithRequestID(requestID).
					WithMessageType("control_response")
			}

			return servers, nil
		case ControlErrorResponse:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("McpServerStatus request failed: %s", r.Error), nil).
				WithSessionID(q.sessionID).
				WithRequestID(requestID).
				WithMessageType("control_response")
		default:
			return nil, clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError, fmt.Sprintf("unexpected control response type: %T", r), nil).
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

// GetServerInfo returns the initialization result stored during Initialize.
func (q *queryImpl) GetServerInfo() (map[string]any, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.initializationResult == nil {
		return nil, clauderrs.NewClientError(clauderrs.ErrCodeInvalidState, "query not initialized", nil).
			WithSessionID(q.sessionID)
	}

	return q.initializationResult, nil
}

// Initialize sends initialize control request and stores the response.
// This should be called if bidirectional control protocol is needed.
func (q *queryImpl) Initialize(ctx context.Context) (map[string]any, error) {
	// Build hooks configuration from opts.Hooks
	var hooksConfig map[string]JSONValue
	if len(q.opts.Hooks) > 0 {
		hooksConfig = make(map[string]JSONValue)

		for event, matchers := range q.opts.Hooks {
			if len(matchers) == 0 {
				continue
			}

			// Build array of hook matchers for this event
			matcherConfigs := make([]map[string]any, 0, len(matchers))
			for _, matcher := range matchers {
				// Register each callback and collect their IDs
				callbackIDs := make([]string, 0, len(matcher.Hooks))
				for _, callback := range matcher.Hooks {
					callbackID := fmt.Sprintf("hook_%d", q.nextCallbackID)
					q.nextCallbackID++
					q.hookCallbacks[callbackID] = callback
					callbackIDs = append(callbackIDs, callbackID)
				}

				// Build matcher config
				matcherConfig := map[string]any{
					"hookCallbackIds": callbackIDs,
				}
				if matcher.Matcher != nil {
					matcherConfig["matcher"] = *matcher.Matcher
				}
				matcherConfigs = append(matcherConfigs, matcherConfig)
			}

			// Marshal to JSONValue
			matcherBytes, err := json.Marshal(matcherConfigs)
			if err != nil {
				return nil, clauderrs.NewProtocolError(
					clauderrs.ErrCodeMessageParseFailed,
					fmt.Sprintf("failed to marshal hook matchers for event %s", event),
					err,
				).
					WithSessionID(q.sessionID).
					WithMessageType("initialize")
			}
			hooksConfig[string(event)] = matcherBytes
		}
	}

	resp, err := q.sendControlRequest(ctx, SDKControlInitializeRequest{
		Hooks: hooksConfig,
	})
	if err != nil {
		return nil, err
	}

	q.mu.Lock()
	q.initializationResult = resp
	q.mu.Unlock()

	return resp, nil
}

// QueryFunc creates a new query session.
func QueryFunc(prompt string, opts *Options) (Query, error) {
	return newQueryImpl(prompt, opts)
}
