## Phase 6: Testing & Documentation
Priority: Critical
### 6.1 Unit Tests
Priority: Critical
Testing Strategy:
Unit tests verify domain logic and adapters in isolation without external dependencies.
Tools & Framework:
- Standard library `testing` package
- Table-driven tests for comprehensive coverage
- `testify/assert` for readable assertions (optional)
- Mock implementations of ports for testing domain services
Test Structure:
```go
// Domain service tests (no infrastructure dependencies)
// pkg/claude/querying/service_test.go
package querying_test

import (
	"context"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/querying"
	"testing"
)

// Mock transport implementing ports.Transport
type mockTransport struct {
	connectErr error
	messages   []map[string]any
}

func (m *mockTransport) Connect(ctx context.Context) error            { return m.connectErr }
func (m *mockTransport) Write(ctx context.Context, data string) error { return nil }

// ... implement other methods
func TestService_Execute(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		wantErr   bool
		setupMock func(*mockTransport)
	}{
		{
			name:    "successful query",
			prompt:  "test query",
			wantErr: false,
			setupMock: func(m *mockTransport) {
				m.messages = []map[string]any{
					{"type": "assistant", "message": map[string]any{"content": "response"}},
				}
			},
		},
		// ... more test cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{}
			if tt.setupMock != nil {
				tt.setupMock(transport)
			}
			protocol := &mockProtocol{}
			parser := &mockParser{}
			hooks := hooking.NewService(nil)
			perms := permissions.NewService(nil)
			mcpServers := make(map[string]ports.MCPServer)
			svc := querying.NewService(transport, protocol, parser, hooks, perms, mcpServers)
			msgCh, errCh := svc.Execute(context.Background(), tt.prompt, &options.AgentOptions{})

			// Verify results - expect at least one message
			select {
			case msg := <-msgCh:
				if msg == nil {
					t.Fatal("expected non-nil message")
				}
			case err := <-errCh:
				if tt.wantErr && err != nil {
					return // Expected error
				}
				if !tt.wantErr && err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
```
Message Parsing Tests:
```go
// adapters/parse/parser_test.go
package parse_test

import (
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    messages.Message
		wantErr bool
	}{
		{
			name: "parse assistant message",
			input: map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"model": "claude-sonnet-4",
					"content": []any{
						map[string]any{"type": "text", "text": "Hello"},
					},
				},
			},
			want: &messages.AssistantMessage{
				Model: "claude-sonnet-4",
				Content: []messages.ContentBlock{
					messages.TextBlock{Text: "Hello"},
				},
			},
		},
		// ... more cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := parse.NewAdapter()
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Compare got with want
			if !tt.wantErr && got == nil {
				t.Fatal("expected non-nil message")
			}
		})
	}
}
```

### 6.1.1 Hermetic CLI Testing (Restricted Environments)

Priority: Critical

**Problem:** Integration tests typically require the Claude CLI to be installed and an API key configured. This creates issues for:
- CI/CD environments without network access
- Developer machines without CLI installed
- Air-gapped or restricted environments
- Fast, repeatable unit tests

**Solution:** Use test doubles, record/replay, and fake transports to test CLI-dependent code without requiring the actual CLI.

#### Strategy 1: Fake Transport (Recommended for Unit Tests)

Create a fake transport that simulates CLI behavior without spawning processes:

```go
// pkg/claude/internal/testutil/fake_transport.go
package testutil

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// FakeTransport simulates CLI transport behavior for testing
type FakeTransport struct {
	mu            sync.Mutex
	responses     []map[string]any
	responseIndex int
	isConnected   bool
	writeHistory  []string
	simulateError error
}

func NewFakeTransport() *FakeTransport {
	return &FakeTransport{
		responses:    make([]map[string]any, 0),
		writeHistory: make([]string, 0),
	}
}

// QueueResponse adds a response to be returned by ReadMessages
func (f *FakeTransport) QueueResponse(msg map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses = append(f.responses, msg)
}

// SimulateError causes the next operation to fail
func (f *FakeTransport) SimulateError(err error) {
	f.simulateError = err
}

func (f *FakeTransport) Connect(ctx context.Context) error {
	if f.simulateError != nil {
		return f.simulateError
	}
	f.isConnected = true
	return nil
}

func (f *FakeTransport) Write(ctx context.Context, data string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.simulateError != nil {
		return f.simulateError
	}
	f.writeHistory = append(f.writeHistory, data)
	return nil
}

func (f *FakeTransport) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, len(f.responses))
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		if f.simulateError != nil {
			errCh <- f.simulateError
			return
		}

		f.mu.Lock()
		defer f.mu.Unlock()

		for _, msg := range f.responses {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return msgCh, errCh
}

func (f *FakeTransport) EndInput() error       { return nil }
func (f *FakeTransport) Close() error          { return nil }
func (f *FakeTransport) IsReady() bool         { return f.isConnected }

// Test helpers
func (f *FakeTransport) GetWriteHistory() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.writeHistory...)
}

var _ ports.Transport = (*FakeTransport)(nil)
```

**Usage in tests:**

```go
func TestQueryWithFakeTransport(t *testing.T) {
	fake := testutil.NewFakeTransport()

	// Queue expected responses
	fake.QueueResponse(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{"type": "text", "text": "2+2 equals 4"},
			},
		},
	})
	fake.QueueResponse(map[string]any{
		"type":    "result",
		"subtype": "success",
	})

	// Create service with fake transport
	svc := querying.NewService(fake, protocol, parser, hooks, perms, mcpServers)
	msgCh, errCh := svc.Execute(context.Background(), "What is 2+2?", nil)

	// Assert expected behavior
	var gotAssistantMsg bool
	for msg := range msgCh {
		if assistantMsg, ok := msg.(messages.AssistantMessage); ok {
			gotAssistantMsg = true
			if len(assistantMsg.Content) == 0 {
				t.Fatal("expected non-empty content")
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !gotAssistantMsg {
		t.Fatal("expected assistant message")
	}

	// Verify what was sent to CLI
	history := fake.GetWriteHistory()
	if len(history) == 0 {
		t.Fatal("expected write operations")
	}
}
```

#### Strategy 2: Record/Replay (Golden Files)

Record real CLI interactions and replay them for deterministic tests:

```go
// pkg/claude/internal/testutil/recording_transport.go
package testutil

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

// RecordingTransport wraps a real transport and records all interactions
type RecordingTransport struct {
	real      ports.Transport
	recording *Recording
}

type Recording struct {
	Writes    []string           `json:"writes"`
	Responses []map[string]any   `json:"responses"`
}

func NewRecordingTransport(real ports.Transport) *RecordingTransport {
	return &RecordingTransport{
		real:      real,
		recording: &Recording{},
	}
}

func (r *RecordingTransport) Write(ctx context.Context, data string) error {
	r.recording.Writes = append(r.recording.Writes, data)
	return r.real.Write(ctx, data)
}

func (r *RecordingTransport) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	realMsgCh, realErrCh := r.real.ReadMessages(ctx)
	msgCh := make(chan map[string]any)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for msg := range realMsgCh {
			r.recording.Responses = append(r.recording.Responses, msg)
			msgCh <- msg
		}

		if err := <-realErrCh; err != nil {
			errCh <- err
		}
	}()

	return msgCh, errCh
}

// SaveRecording writes the recording to a file
func (r *RecordingTransport) SaveRecording(path string) error {
	data, err := json.MarshalIndent(r.recording, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// ReplayTransport replays a recorded interaction
type ReplayTransport struct {
	recording *Recording
	writeIdx  int
}

func NewReplayTransport(recordingPath string) (*ReplayTransport, error) {
	data, err := ioutil.ReadFile(recordingPath)
	if err != nil {
		return nil, err
	}

	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}

	return &ReplayTransport{recording: &rec}, nil
}

func (r *ReplayTransport) Connect(ctx context.Context) error { return nil }
func (r *ReplayTransport) IsReady() bool                     { return true }
func (r *ReplayTransport) Close() error                      { return nil }
func (r *ReplayTransport) EndInput() error                   { return nil }

func (r *ReplayTransport) Write(ctx context.Context, data string) error {
	// Verify writes match recording
	if r.writeIdx >= len(r.recording.Writes) {
		return fmt.Errorf("unexpected write (recording has %d writes)", len(r.recording.Writes))
	}
	if r.recording.Writes[r.writeIdx] != data {
		return fmt.Errorf("write mismatch at index %d", r.writeIdx)
	}
	r.writeIdx++
	return nil
}

func (r *ReplayTransport) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, len(r.recording.Responses))
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for _, msg := range r.recording.Responses {
			msgCh <- msg
		}
	}()

	return msgCh, errCh
}

var _ ports.Transport = (*ReplayTransport)(nil)
```

**Usage:**

```go
func TestWithRecording(t *testing.T) {
	// To record (run once with real CLI):
	// transport := cli.NewAdapter(opts)
	// recorder := testutil.NewRecordingTransport(transport)
	// ... run test ...
	// recorder.SaveRecording("testdata/simple_query.json")

	// To replay (no CLI required):
	replay, err := testutil.NewReplayTransport("testdata/simple_query.json")
	if err != nil {
		t.Fatalf("failed to load recording: %v", err)
	}

	svc := querying.NewService(replay, protocol, parser, hooks, perms, mcpServers)
	msgCh, errCh := svc.Execute(context.Background(), "What is 2+2?", nil)

	// Test assertions...
}
```

#### Strategy 3: Build Tag Separation

Use build tags to separate integration tests from unit tests:

```go
//go:build !integration
// +build !integration

// This file runs in normal `go test` (no CLI required)
func TestServiceLogic_Unit(t *testing.T) {
	fake := testutil.NewFakeTransport()
	// ... test with fake
}
```

```go
//go:build integration
// +build integration

// This file only runs with `go test -tags=integration` (requires CLI)
func TestRealCLI_Integration(t *testing.T) {
	transport := cli.NewAdapter(opts)
	// ... test with real CLI
}
```

#### Running Tests in Restricted Environments

```bash
# Unit tests only (no CLI required)
go test ./pkg/claude/...

# Integration tests (requires CLI)
go test -tags=integration ./tests/integration/...

# Run all tests with coverage (skip integration if CLI unavailable)
go test -cover ./...

# CI/CD: Use fake transports by default
go test -v -race ./pkg/claude/...
```

Adapter Tests (with mocked subprocess):
```go
// adapters/cli/transport_test.go
package cli_test

import (
	"context"
	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/options"
	"testing"
)

func TestAdapter_FindCLI(t *testing.T) {
	// Test CLI discovery logic
	// Set up test PATH environment
	tests := []struct {
		name     string
		pathEnv  string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "finds claude in PATH",
			pathEnv:  "/usr/local/bin:/usr/bin",
			wantPath: "/usr/local/bin/claude",
			wantErr:  false,
		},
		{
			name:     "returns error when not found",
			pathEnv:  "/empty/path",
			wantPath: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock PATH
			oldPath := os.Getenv("PATH")
			os.Setenv("PATH", tt.pathEnv)
			defer os.Setenv("PATH", oldPath)

			adapter := cli.NewAdapter(nil)
			gotPath, err := adapter.FindCLI()

			if (err != nil) != tt.wantErr {
				t.Errorf("FindCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotPath != tt.wantPath {
				t.Errorf("FindCLI() path = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestAdapter_BuildCommand(t *testing.T) {
	tests := []struct {
		name    string
		opts    *options.AgentOptions
		want    []string
		wantErr bool
	}{
		{
			name: "basic command",
			opts: &options.AgentOptions{
				Model: stringPtr("claude-sonnet-4"),
			},
			want: []string{"claude", "--output-format", "stream-json", "--model", "claude-sonnet-4"},
		},
		{
			name: "with max turns",
			opts: &options.AgentOptions{
				Model:    stringPtr("claude-sonnet-4"),
				MaxTurns: intPtr(5),
			},
			want: []string{"claude", "--output-format", "stream-json", "--model", "claude-sonnet-4", "--max-turns", "5"},
		},
		// ... more cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := cli.NewAdapter(tt.opts)
			got, err := adapter.BuildCommand()

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare command slices
			if len(got) != len(tt.want) {
				t.Errorf("BuildCommand() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildCommand()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
```

SDK MCP Server Adapter Tests:

**IMPORTANT:** These tests reflect the current manual routing implementation, NOT an in-memory transport abstraction.

```go
// adapters/mcp/sdk_server_test.go
package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/adapters/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestSDKServerAdapter_ManualRouting tests the current manual routing implementation
// NOTE: This does NOT use an in-memory transport because the Go MCP SDK lacks
// transport abstractions (similar to Python SDK limitation)
func TestSDKServerAdapter_ManualRouting(t *testing.T) {
	// Create MCP server instance
	server := mcpsdk.NewServer("test", "1.0")

	// Create adapter - wraps server for manual JSON-RPC routing
	adapter, err := mcp.NewSDKServerAdapter("test", server)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	// Test initialize method routing
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "0.1.0",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	response, err := adapter.HandleRequest(ctx, initRequest)
	if err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}

	// Verify JSON-RPC response structure
	if response["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", response["jsonrpc"])
	}

	if response["id"] != 1 {
		t.Errorf("expected id 1, got %v", response["id"])
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatal("expected result object in response")
	}

	if result["protocolVersion"] != "0.1.0" {
		t.Errorf("expected protocolVersion 0.1.0, got %v", result["protocolVersion"])
	}
}

func TestSDKServerAdapter_ToolListRouting(t *testing.T) {
	server := mcpsdk.NewServer("test", "1.0")

	// Add test tool
	server.AddTool(&mcpsdk.Tool{
		Name:        "add",
		Description: "Add two numbers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "number"},
				"b": map[string]any{"type": "number"},
			},
		},
	})

	adapter, _ := mcp.NewSDKServerAdapter("test", server)

	// Test tools/list routing
	listRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}

	response, err := adapter.HandleRequest(context.Background(), listRequest)
	if err != nil {
		t.Fatalf("tools/list request failed: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatal("expected result in response")
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("expected tools array in result")
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0].(map[string]any)
	if tool["name"] != "add" {
		t.Errorf("expected tool name 'add', got %v", tool["name"])
	}
}

func TestSDKServerAdapter_ToolCallRouting(t *testing.T) {
	server := mcpsdk.NewServer("test", "1.0")

	// Add test tool with handler
	type AddArgs struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	handlerCalled := false
	handler := func(ctx context.Context, req *mcpsdk.CallToolRequest, args AddArgs) (*mcpsdk.CallToolResult, struct{ Sum int }, error) {
		handlerCalled = true
		return nil, struct{ Sum int }{Sum: args.A + args.B}, nil
	}

	// Register tool (using SDK registration method)
	server.AddTool(&mcpsdk.Tool{
		Name:        "add",
		Description: "Add two numbers",
	})
	server.RegisterToolHandler("add", handler)

	adapter, _ := mcp.NewSDKServerAdapter("test", server)

	// Test tools/call routing
	callRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "add",
			"arguments": map[string]any{
				"a": 5,
				"b": 3,
			},
		},
	}

	response, err := adapter.HandleRequest(context.Background(), callRequest)
	if err != nil {
		t.Fatalf("tools/call request failed: %v", err)
	}

	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}

	// Verify JSON-RPC response
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatal("expected result in response")
	}

	// Check content array structure
	content, ok := result["content"].([]any)
	if !ok {
		t.Fatal("expected content array in result")
	}

	if len(content) == 0 {
		t.Fatal("expected non-empty content array")
	}

	// Verify the sum value in response
	contentItem := content[0].(map[string]any)
	if contentItem["type"] != "text" {
		t.Errorf("expected type 'text', got %v", contentItem["type"])
	}

	// Text should contain the result
	text := contentItem["text"].(string)
	if text == "" {
		t.Error("expected non-empty text result")
	}
}

// TestSDKServerAdapter_ErrorHandling tests error scenarios in manual routing
func TestSDKServerAdapter_ErrorHandling(t *testing.T) {
	server := mcpsdk.NewServer("test", "1.0")
	adapter, _ := mcp.NewSDKServerAdapter("test", server)

	tests := []struct {
		name        string
		request     map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "missing method",
			request: map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
			},
			wantErr:     true,
			errContains: "method",
		},
		{
			name: "unknown method",
			request: map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "unknown/method",
			},
			wantErr:     true,
			errContains: "unknown method",
		},
		{
			name: "malformed params",
			request: map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params":  "not an object",
			},
			wantErr:     true,
			errContains: "params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := adapter.HandleRequest(context.Background(), tt.request)

			if tt.wantErr {
				// Should return JSON-RPC error response
				if err == nil && response["error"] == nil {
					t.Fatal("expected error or error response")
				}

				if err != nil && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 s[1:len(s)-1] != s && contains(s[1:], substr)))
}
```

Run Unit Tests:
```bash
go test -v ./pkg/claude/...
go test -race ./pkg/claude/...  # Check for race conditions
go test -cover ./pkg/claude/... # Check coverage
```
### 6.2 Integration Tests
Priority: High
Testing Strategy:
Integration tests verify the SDK works with the actual Claude CLI.
Prerequisites:
- Claude CLI must be installed: `npm install -g @anthropic-ai/claude-code`
- API key must be set in environment or config
- Tests should be skippable if CLI is not available
Test Structure:
```go
// tests/integration/query_test.go
//go:build integration
// +build integration
package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func TestMain(m *testing.M) {
	// Check if Claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("Skipping integration tests: claude CLI not found")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func TestQuery_BasicInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opts := &options.AgentOptions{
		MaxTurns: intPtr(1),
	}
	msgCh, errCh := claude.Query(ctx, "What is 2+2?", opts, nil)
	var gotResponse bool
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if !gotResponse {
					t.Fatal("no response received")
				}
				return
			}
			if assistantMsg, ok := msg.(messages.AssistantMessage); ok {
				gotResponse = true
				t.Logf("Received: %+v", assistantMsg)
				// Assert content is not empty
				if len(assistantMsg.Content) == 0 {
					t.Error("expected non-empty content in assistant message")
				}
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("error: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}
}
func TestStreamingClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()
	client := claude.NewClient(&options.AgentOptions{})
	if err := client.Connect(ctx, nil); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()
	if err := client.SendMessage(ctx, "Hello"); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	msgCh, errCh := client.ReceiveMessages(ctx)

	timeout := time.After(30 * time.Second)
	var gotMessage bool

	select {
	case msg, ok := <-msgCh:
		if !ok {
			t.Fatal("message channel closed without receiving message")
		}
		gotMessage = true
		t.Logf("Received: %+v", msg)
		// Assert message is valid
		if msg == nil {
			t.Error("received nil message")
		}
	case err := <-errCh:
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	case <-timeout:
		t.Fatal("timeout waiting for message")
	}

	if !gotMessage {
		t.Fatal("expected to receive at least one message")
	}
}
func TestSDKMCPServer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()

	// Create SDK MCP server with calculator tools
	server := claude.NewMCPServer("calculator", "1.0")

	type MathArgs struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	type MathResult struct {
		Result int `json:"result"`
	}

	addHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, args MathArgs) (*mcpsdk.CallToolResult, MathResult, error) {
		return nil, MathResult{Result: args.A + args.B}, nil
	}

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "add",
		Description: "Add two numbers",
	}, addHandler)

	// Configure client with SDK MCP server
	opts := &options.AgentOptions{
		MCPServers: map[string]options.MCPServerConfig{
			"calculator": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "calculator",
				Instance: server,
			},
		},
		AllowedTools: []options.BuiltinTool{options.ToolMcp},
	}

	client := claude.NewClient(opts, nil, nil)
	if err := client.Connect(ctx, nil); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	// Test that Claude can use the calculator tool
	if err := client.SendMessage(ctx, "Use the calculator to add 5 and 3"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	msgCh, errCh := client.ReceiveMessages(ctx)
	timeout := time.After(60 * time.Second)
	var foundToolUse bool

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if !foundToolUse {
					t.Fatal("expected tool use message but channel closed")
				}
				return
			}
			// Look for tool use in message
			if assistantMsg, ok := msg.(messages.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if toolUse, ok := block.(messages.ToolUseBlock); ok {
						if toolUse.Name == "add" {
							foundToolUse = true
							t.Logf("Tool used successfully: %+v", toolUse)
							// Assert tool has valid arguments
							if toolUse.Arguments == nil {
								t.Error("expected non-nil tool arguments")
							}
						}
					}
				}
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("error: %v", err)
			}
		case <-timeout:
			t.Fatal("timeout waiting for tool use")
		}
	}
}
```
Run Integration Tests:
```bash
# Run with integration tag
go test -tags=integration -v ./tests/integration/...
# Run with race detector and coverage
go test -tags=integration -race -coverprofile=coverage.txt ./tests/integration/...
```
### 6.3 Test Fixtures & Mocking
Shared Mocks:
Create reusable mocks in a dedicated package:
```go
// pkg/claude/internal/testutil/mocks.go
package testutil

import (
	"context"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

type MockTransport struct {
	ConnectFunc      func(context.Context) error
	WriteFunc        func(context.Context, string) error
	ReadMessagesFunc func(context.Context) (<-chan map[string]any, <-chan error)
	EndInputFunc     func() error
	CloseFunc        func() error
	IsReadyFunc      func() bool
}

func (m *MockTransport) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *MockTransport) Write(ctx context.Context, data string) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, data)
	}
	return nil
}

func (m *MockTransport) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	if m.ReadMessagesFunc != nil {
		return m.ReadMessagesFunc(ctx)
	}
	msgCh := make(chan map[string]any)
	errCh := make(chan error)
	close(msgCh)
	close(errCh)
	return msgCh, errCh
}

func (m *MockTransport) EndInput() error {
	if m.EndInputFunc != nil {
		return m.EndInputFunc()
	}
	return nil
}

func (m *MockTransport) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockTransport) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}

// Compile-time interface check
var _ ports.Transport = (*MockTransport)(nil)
```
Test Data:
```go
// pkg/claude/internal/testutil/fixtures.go
package testutil

var (
	AssistantMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{"type": "text", "text": "Hello"},
			},
		},
	}
	ResultMessageJSON = map[string]any{
		"type":        "result",
		"subtype":     "success",
		"duration_ms": 1234,
		"num_turns":   1,
		"session_id":  "test-session",
	}
	ToolUseMessageJSON = map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4",
			"content": []any{
				map[string]any{
					"type": "tool_use",
					"id":   "tool_123",
					"name": "bash",
					"input": map[string]any{
						"command": "echo hello",
					},
				},
			},
		},
	}
)
```
### 6.5 Examples
Priority: High
Create comprehensive, runnable examples:
Quick Start Example:
```go
// cmd/examples/quickstart/main.go
package main

import (
	"context"
	"fmt"
	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"log"
)

func main() {
	ctx := context.Background()
	msgCh, errCh := claude.Query(ctx, "What is 2 + 2?", nil, nil)
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			if assistantMsg, ok := msg.(messages.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(messages.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		case err := <-errCh:
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}
}
```
Examples to Create:
1. `quickstart/` - Basic query
2. `streaming/` - Bidirectional conversation
3. `hooks/` - Custom hooks
4. `mcp/calculator/` - SDK MCP server with math tools (see Phase 5b)
5. `mcp/external/` - External MCP server integration
6. `permissions/` - Permission callbacks
7. `tools/` - Tool filtering
8. `integration/` - Combined hooks + MCP + permissions (see Phase 5 summary)
### 6.6 Documentation
Priority: High
- Comprehensive README.md with architecture diagram
- API documentation with godoc comments
- Migration guide from Python SDK
- Architecture documentation (hexagonal structure)
- Hook development guide
- MCP server development guide

---

## Linting Compliance Notes

### Test File Size Strategy (175 line limit)

**Critical pattern: Use table-driven tests extensively**

Test files can easily exceed 175 lines. Strategy:

**Pattern 1: Separate Fixtures**
```
tests/
├── fixtures.go         # Shared test data (100 lines)
├── service_test.go     # Table-driven tests (120 lines)
└── helpers.go          # Test utilities (80 lines)
```

**Pattern 2: One Test File Per Feature**
```
querying/
├── service.go
├── execute_test.go     # Test Execute method (100 lines)
├── routing_test.go     # Test routing logic (90 lines)
└── fixtures_test.go    # Test fixtures (60 lines)
```

**Pattern 3: Table-Driven Tests**
```go
// Keeps test files under 175 lines
func TestParseMessage(t *testing.T) {
    tests := []struct {
        name     string
        input    map[string]any
        want     messages.Message
        wantErr  bool
    }{
        {"case1", fixture1, expected1, false},
        {"case2", fixture2, expected2, true},
        // ... 50 test cases in compact form
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parse.ParseMessage(tt.input)
            // Assertions
        })
    }
}
```

### Complexity in Tests

**Test functions must also follow 25-line limit:**
- Extract setup functions
- Extract assertion helpers
- Use testify/assert for concise assertions

**Example compliant test:**
```go
func TestServiceExecute(t *testing.T) {
    svc, mocks := setupService(t) // Extracted

    result, err := svc.Execute(context.Background(), "test")

    assertNoError(t, err)          // Extracted
    assertValidResult(t, result)   // Extracted
}
```

### Checklist

- [ ] All test files under 175 lines
- [ ] Use table-driven tests where applicable
- [ ] Shared fixtures in separate file
- [ ] Test helpers in separate file
- [ ] Test functions under 25 lines
- [ ] Mock setup functions extracted
- [ ] Assertion helpers extracted
