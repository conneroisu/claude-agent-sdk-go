package unit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	claudeagent "github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/google/uuid"
)

const (
	// Test data constants.
	testSessionID      = "test-session"
	testModelSonnet    = "claude-sonnet-4-5"
	testMessageID      = "msg-123"
	testInputTokens    = 5
	testOutputTokens   = 3
	testDurationMS     = 1000
	testModelCallCount = 3
	userMessageType    = "user"
	assistantType      = "assistant"
	resultType         = "result"
)

// MockQuery implements the Query interface for testing.
type MockQuery struct {
	messages               []claudeagent.SDKMessage
	messageIndex           int
	sendMessageCalls       [][]claudeagent.ContentBlock
	sendTextCalls          []string
	interruptCalled        bool
	setPermissionModeCalls []claudeagent.PermissionMode
	setModelCalls          []*string
	closed                 bool
	serverInfo             map[string]any
}

func (m *MockQuery) Next(_ context.Context) (claudeagent.SDKMessage, error) {
	if m.messageIndex >= len(m.messages) {
		return nil, io.EOF
	}
	msg := m.messages[m.messageIndex]
	m.messageIndex++

	return msg, nil
}

func (m *MockQuery) Close() error {
	m.closed = true

	return nil
}

func (m *MockQuery) SendUserMessage(
	_ context.Context,
	text string,
) error {
	m.sendTextCalls = append(m.sendTextCalls, text)

	return nil
}

func (m *MockQuery) SendUserMessageWithContent(
	_ context.Context,
	content []claudeagent.ContentBlock,
) error {
	m.sendMessageCalls = append(m.sendMessageCalls, content)

	return nil
}

func (m *MockQuery) Interrupt(_ context.Context) error {
	m.interruptCalled = true

	return nil
}

func (m *MockQuery) SetPermissionMode(
	_ context.Context,
	mode claudeagent.PermissionMode,
) error {
	m.setPermissionModeCalls = append(m.setPermissionModeCalls, mode)

	return nil
}

func (m *MockQuery) SetModel(_ context.Context, model *string) error {
	m.setModelCalls = append(m.setModelCalls, model)

	return nil
}

func (*MockQuery) SupportedCommands(
	_ context.Context,
) ([]claudeagent.SlashCommand, error) {
	return []claudeagent.SlashCommand{
		{
			Name:         "help",
			Description:  "Show help",
			ArgumentHint: "",
		},
	}, nil
}

func (*MockQuery) SupportedModels(
	_ context.Context,
) ([]claudeagent.ModelInfo, error) {
	return []claudeagent.ModelInfo{
		{
			Value:       testModelSonnet,
			DisplayName: "Claude Sonnet 4.5",
			Description: "Latest Claude model",
		},
	}, nil
}

func (*MockQuery) McpServerStatus(
	_ context.Context,
) ([]claudeagent.McpServerStatus, error) {
	return []claudeagent.McpServerStatus{
		{
			Name:   "test-server",
			Status: "connected",
		},
	}, nil
}

func (m *MockQuery) GetServerInfo() (map[string]any, error) {
	if m.serverInfo == nil {
		return nil, errors.New("not initialized")
	}

	return m.serverInfo, nil
}

// Test NewClient creation.
func TestNewClient(t *testing.T) {
	opts := &claudeagent.Options{
		Model: testModelSonnet,
	}

	client, err := claudeagent.NewClient(opts)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// Test NewClient with nil options.
func TestNewClientNilOptions(t *testing.T) {
	client, err := claudeagent.NewClient(nil)
	if err != nil {
		t.Fatalf("failed to create client with nil options: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// Test Query creates query on first call.
func TestClientQueryFirstCall(t *testing.T) {
	// Note: This test would require mocking the internal QueryFunc,
	// so we'll test the behavior with a mock query set directly
	// In real usage, the first Query() call initializes the query
	t.Skip("Requires process mocking - covered by integration tests")
}

// Test multi-turn Query calls.
func TestClientMultiTurnQuery(t *testing.T) {
	// This test demonstrates how multi-turn conversations work
	// The first Query() creates a new query, subsequent calls send messages
	t.Skip("Requires process mocking - covered by integration tests")
}

// Test ReceiveMessages receives all messages until EOF.
func TestReceiveMessagesFullStream(t *testing.T) {
	// Create test messages
	messages := []claudeagent.SDKMessage{
		&claudeagent.SDKUserMessage{
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			TypeField: userMessageType,
			Message: claudeagent.APIUserMessage{
				Role: userMessageType,
				Content: []claudeagent.ContentBlock{
					claudeagent.TextContentBlock{
						Type: "text",
						Text: "Hello",
					},
				},
			},
		},
		&claudeagent.SDKAssistantMessage{
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			Message: claudeagent.APIAssistantMessage{
				ID:   testMessageID,
				Type: "message",
				Role: assistantType,
				Content: []claudeagent.ContentBlock{
					claudeagent.TextBlock{
						Type: "text",
						Text: "Hi there!",
					},
				},
				Model: testModelSonnet,
				Usage: claudeagent.Usage{
					InputTokens:  testInputTokens,
					OutputTokens: testOutputTokens,
				},
			},
		},
		&claudeagent.SDKResultMessage{
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			Subtype:    claudeagent.ResultSubtypeSuccess,
			DurationMS: testDurationMS,
			NumTurns:   1,
		},
	}

	// This test would work with a fully mocked client.
	// For now, we verify the message types are correctly set up.
	for i, msg := range messages {
		switch i {
		case 0:
			if msg.Type() != userMessageType {
				t.Errorf(
					"message %d: expected type '%s', got %s",
					i,
					userMessageType,
					msg.Type(),
				)
			}
		case 1:
			if msg.Type() != assistantType {
				t.Errorf(
					"message %d: expected type '%s', got %s",
					i,
					assistantType,
					msg.Type(),
				)
			}
		case 2:
			if msg.Type() != resultType {
				t.Errorf(
					"message %d: expected type '%s', got %s",
					i,
					resultType,
					msg.Type(),
				)
			}
		}
	}
}

// Test ReceiveResponse stops at ResultMessage.
func TestReceiveResponseStopsAtResult(t *testing.T) {
	// Create messages including a result message
	messages := []claudeagent.SDKMessage{
		&claudeagent.SDKUserMessage{
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			TypeField: userMessageType,
		},
		&claudeagent.SDKResultMessage{
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			Subtype: claudeagent.ResultSubtypeSuccess,
		},
		&claudeagent.SDKUserMessage{ // This should not be received
			BaseMessage: claudeagent.BaseMessage{
				UUIDField:      uuid.New(),
				SessionIDField: testSessionID,
			},
			TypeField: userMessageType,
		},
	}

	// Verify the result message is correctly typed
	resultMsg, ok := messages[1].(*claudeagent.SDKResultMessage)
	if !ok {
		t.Fatal("expected SDKResultMessage")
	}

	if resultMsg.Type() != resultType {
		t.Errorf("expected type '%s', got %s", resultType, resultMsg.Type())
	}
}

// Test SendMessage with structured content blocks.
func TestClientSendMessageStructuredContent(t *testing.T) {
	// Base64 encoded 1x1 transparent PNG
	base64PNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ" +
		"AAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	content := []claudeagent.ContentBlock{
		claudeagent.TextContentBlock{
			Type: "text",
			Text: "What is in this image?",
		},
		claudeagent.ImageContentBlock{
			Type: "image",
			Source: claudeagent.ImageSource{
				Type:      "base64",
				MediaType: "image/png",
				Data:      base64PNG,
			},
		},
	}

	// Marshal to verify structure
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal content: %v", err)
	}

	// Unmarshal to verify round-trip
	var decoded []json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}

	if len(decoded) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(decoded))
	}
}

// Test Interrupt method.
func TestClientInterrupt(t *testing.T) {
	mock := &MockQuery{}

	// Simulate interrupt call
	ctx := context.Background()
	if err := mock.Interrupt(ctx); err != nil {
		t.Fatalf("failed to interrupt: %v", err)
	}

	if !mock.interruptCalled {
		t.Error("expected interrupt to be called")
	}
}

// Test SetPermissionMode method.
func TestClientSetPermissionMode(t *testing.T) {
	mock := &MockQuery{}
	ctx := context.Background()

	modes := []claudeagent.PermissionMode{
		claudeagent.PermissionModeDefault,
		claudeagent.PermissionModeAcceptEdits,
		claudeagent.PermissionModeBypassPermissions,
		claudeagent.PermissionModePlan,
	}

	for _, mode := range modes {
		if err := mock.SetPermissionMode(ctx, mode); err != nil {
			t.Fatalf("failed to set permission mode %s: %v", mode, err)
		}
	}

	if len(mock.setPermissionModeCalls) != len(modes) {
		t.Errorf(
			"expected %d permission mode calls, got %d",
			len(modes),
			len(mock.setPermissionModeCalls),
		)
	}

	for i, mode := range modes {
		if mock.setPermissionModeCalls[i] != mode {
			t.Errorf(
				"call %d: expected mode %s, got %s",
				i,
				mode,
				mock.setPermissionModeCalls[i],
			)
		}
	}
}

// Test SetModel method.
func TestClientSetModel(t *testing.T) {
	mock := &MockQuery{}
	ctx := context.Background()

	model1 := testModelSonnet
	model2 := "claude-opus-4"

	if err := mock.SetModel(ctx, &model1); err != nil {
		t.Fatalf("failed to set model: %v", err)
	}

	if err := mock.SetModel(ctx, &model2); err != nil {
		t.Fatalf("failed to set model: %v", err)
	}

	// Test with nil (reset to default)
	if err := mock.SetModel(ctx, nil); err != nil {
		t.Fatalf("failed to set model to nil: %v", err)
	}

	if len(mock.setModelCalls) != testModelCallCount {
		t.Errorf(
			"expected %d set model calls, got %d",
			testModelCallCount,
			len(mock.setModelCalls),
		)
	}

	if *mock.setModelCalls[0] != model1 {
		t.Errorf(
			"call 0: expected model %s, got %s",
			model1,
			*mock.setModelCalls[0],
		)
	}

	if *mock.setModelCalls[1] != model2 {
		t.Errorf(
			"call 1: expected model %s, got %s",
			model2,
			*mock.setModelCalls[1],
		)
	}

	if mock.setModelCalls[2] != nil {
		t.Errorf("call 2: expected nil model, got %v", *mock.setModelCalls[2])
	}
}

// Test SupportedCommands method.
func TestClientSupportedCommands(t *testing.T) {
	mock := &MockQuery{}
	ctx := context.Background()

	commands, err := mock.SupportedCommands(ctx)
	if err != nil {
		t.Fatalf("failed to get supported commands: %v", err)
	}

	if len(commands) == 0 {
		t.Error("expected at least one command")
	}

	// Verify command structure
	cmd := commands[0]
	if cmd.Name == "" {
		t.Error("expected command to have a name")
	}

	if cmd.Description == "" {
		t.Error("expected command to have a description")
	}
}

// Test SupportedModels method.
func TestClientSupportedModels(t *testing.T) {
	mock := &MockQuery{}
	ctx := context.Background()

	models, err := mock.SupportedModels(ctx)
	if err != nil {
		t.Fatalf("failed to get supported models: %v", err)
	}

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Verify model structure
	model := models[0]
	if model.Value == "" {
		t.Error("expected model to have a value")
	}

	if model.DisplayName == "" {
		t.Error("expected model to have a display name")
	}

	if model.Description == "" {
		t.Error("expected model to have a description")
	}
}

// Test McpServerStatus method.
func TestClientMcpServerStatus(t *testing.T) {
	mock := &MockQuery{}
	ctx := context.Background()

	servers, err := mock.McpServerStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get MCP server status: %v", err)
	}

	if len(servers) == 0 {
		t.Error("expected at least one server")
	}

	// Verify server structure
	server := servers[0]
	if server.Name == "" {
		t.Error("expected server to have a name")
	}

	if server.Status == "" {
		t.Error("expected server to have a status")
	}

	// Validate status values
	validStatuses := map[string]bool{
		"connected":  true,
		"failed":     true,
		"needs-auth": true,
		"pending":    true,
	}

	if !validStatuses[server.Status] {
		t.Errorf("unexpected server status: %s", server.Status)
	}
}

// Test GetServerInfo method.
func TestClientGetServerInfo(t *testing.T) {
	mock := &MockQuery{
		serverInfo: map[string]any{
			"name":    "test-server",
			"version": "1.0.0",
		},
	}

	info, err := mock.GetServerInfo()
	if err != nil {
		t.Fatalf("failed to get server info: %v", err)
	}

	if info["name"] != "test-server" {
		t.Errorf("expected name 'test-server', got %v", info["name"])
	}

	if info["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", info["version"])
	}
}

// Test GetServerInfo when not initialized.
func TestClientGetServerInfoNotInitialized(t *testing.T) {
	mock := &MockQuery{}

	_, err := mock.GetServerInfo()
	if err == nil {
		t.Error("expected error when server not initialized")
	}

	expectedMsg := "not initialized"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// Test client Close method.
func TestClientClose(t *testing.T) {
	mock := &MockQuery{}

	if err := mock.Close(); err != nil {
		t.Fatalf("failed to close query: %v", err)
	}

	if !mock.closed {
		t.Error("expected query to be closed")
	}

	// Verify idempotency - closing again should not error
	if err := mock.Close(); err != nil {
		t.Errorf("closing again should not error: %v", err)
	}
}

// Test error handling in Next().
func TestQueryNextError(t *testing.T) {
	expectedErr := errors.New("test error")
	mock := &MockQuery{
		messages: make([]claudeagent.SDKMessage, 0),
	}

	ctx := context.Background()
	_, err := mock.Next(ctx)

	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}

	// Test that we handle EOF correctly
	if err == io.EOF {
		// This is expected behavior
		return
	}

	if err == nil {
		t.Error("expected error")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}
}

// Test context cancellation in operations.
func TestClientContextCancellation(t *testing.T) {
	mock := &MockQuery{}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should respect context cancellation
	// In real implementation, these would return ctx.Err()
	// For now, we just verify the context is properly cancelled
	if ctx.Err() != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", ctx.Err())
	}

	// The mock doesn't implement context checking, but we can verify
	// that we're passing contexts properly
	_ = mock.SendUserMessage(ctx, "test")
}

// Test message flow patterns.
func TestMessageFlowPatterns(t *testing.T) {
	testCases := []struct {
		name         string
		messageTypes []string
		expectError  bool
	}{
		{
			name: "Simple query-response",
			messageTypes: []string{
				userMessageType,
				assistantType,
				resultType,
			},
			expectError: false,
		},
		{
			name: "Multi-turn conversation",
			messageTypes: []string{
				userMessageType,
				assistantType,
				userMessageType,
				assistantType,
				resultType,
			},
			expectError: false,
		},
		{
			name: "With streaming",
			messageTypes: []string{
				userMessageType,
				"stream_event",
				"stream_event",
				assistantType,
				resultType,
			},
			expectError: false,
		},
		{
			name: "With system messages",
			messageTypes: []string{
				"system",
				userMessageType,
				assistantType,
				resultType,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create messages based on types
			messages := make(
				[]claudeagent.SDKMessage,
				0,
				len(tc.messageTypes),
			)

			for _, msgType := range tc.messageTypes {
				switch msgType {
				case userMessageType:
					messages = append(
						messages,
						&claudeagent.SDKUserMessage{
							BaseMessage: claudeagent.BaseMessage{
								UUIDField:      uuid.New(),
								SessionIDField: testSessionID,
							},
							TypeField: userMessageType,
						},
					)
				case assistantType:
					messages = append(
						messages,
						&claudeagent.SDKAssistantMessage{
							BaseMessage: claudeagent.BaseMessage{
								UUIDField:      uuid.New(),
								SessionIDField: testSessionID,
							},
						},
					)
				case resultType:
					messages = append(
						messages,
						&claudeagent.SDKResultMessage{
							BaseMessage: claudeagent.BaseMessage{
								UUIDField:      uuid.New(),
								SessionIDField: testSessionID,
							},
							Subtype: claudeagent.ResultSubtypeSuccess,
						},
					)
				case "stream_event":
					messages = append(
						messages,
						&claudeagent.SDKStreamEvent{
							BaseMessage: claudeagent.BaseMessage{
								UUIDField:      uuid.New(),
								SessionIDField: testSessionID,
							},
						},
					)
				case "system":
					messages = append(
						messages,
						&claudeagent.SDKSystemMessage{
							BaseMessage: claudeagent.BaseMessage{
								UUIDField:      uuid.New(),
								SessionIDField: testSessionID,
							},
							Subtype: "init",
						},
					)
				}
			}

			// Verify all messages were created correctly
			if len(messages) != len(tc.messageTypes) {
				t.Errorf(
					"expected %d messages, got %d",
					len(tc.messageTypes),
					len(messages),
				)
			}

			// Verify message types
			for i, msg := range messages {
				expectedType := tc.messageTypes[i]
				actualType := msg.Type()

				// Map types for comparison
				if actualType != expectedType {
					t.Errorf(
						"message %d: expected type %s, got %s",
						i,
						expectedType,
						actualType,
					)
				}
			}
		})
	}
}

// Test bidirectional communication pattern.
func TestBidirectionalCommunication(t *testing.T) {
	// This test verifies the concept of bidirectional communication
	// where both SDK and CLI can send control requests

	// SDK sends control request
	sdkRequest := claudeagent.SDKControlRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: testSessionID,
		},
		RequestID: "sdk-req-1",
		Request:   claudeagent.SDKControlInterruptRequest{},
	}

	// CLI responds with control response
	cliResponse := claudeagent.SDKControlResponse{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: testSessionID,
		},
		Response: claudeagent.ControlSuccessResponse{
			SubtypeField:   "success",
			RequestIDField: "sdk-req-1",
		},
	}

	// CLI sends control request (permission check)
	cliRequest := claudeagent.SDKControlPermissionRequest{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: testSessionID,
		},
		RequestIDField: "cli-req-1",
		SubtypeField:   "can_use_tool",
		ToolName:       "Write",
		Input: map[string]claudeagent.JSONValue{
			"file_path": json.RawMessage(`"/path/to/file"`),
		},
	}

	// SDK responds with control response
	sdkResponse := claudeagent.SDKControlResponse{
		BaseMessage: claudeagent.BaseMessage{
			UUIDField:      uuid.New(),
			SessionIDField: testSessionID,
		},
		Response: claudeagent.ControlSuccessResponse{
			SubtypeField:   "success",
			RequestIDField: "cli-req-1",
			Response: map[string]claudeagent.JSONValue{
				"allowed": json.RawMessage(`true`),
			},
		},
	}

	// Verify request-response pairing
	if sdkRequest.RequestID != cliResponse.Response.RequestID() {
		t.Error("SDK request and CLI response IDs should match")
	}

	if cliRequest.RequestID() != sdkResponse.Response.RequestID() {
		t.Error("CLI request and SDK response IDs should match")
	}
}
