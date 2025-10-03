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
			svc := querying.NewService(transport, protocol)
			msgCh, errCh := svc.Execute(context.Background(), tt.prompt, &options.AgentOptions{})
			// Verify results
		})
	}
}
```
Message Parsing Tests:
```go
// internal/parse/parser_test.go
package parse_test

import (
	"github.com/conneroisu/claude/pkg/claude/internal/parse"
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
			got, err := parse.ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Compare got with want
		})
	}
}
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
		// ... more cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := cli.NewAdapter(tt.opts)
			got, err := adapter.BuildCommand()
			// Compare got with want
		})
	}
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
	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"os"
	"testing"
	"time"
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
	msgCh, errCh := claude.Query(ctx, "What is 2+2?", opts)
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
			if assistantMsg, ok := msg.(*messages.AssistantMessage); ok {
				gotResponse = true
				t.Logf("Received: %+v", assistantMsg)
			}
		case err := <-errCh:
			t.Fatalf("error: %v", err)
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
	select {
	case msg := <-msgCh:
		t.Logf("Received: %+v", msg)
	case err := <-errCh:
		t.Fatalf("error: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("timeout")
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

// ... implement other methods
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
	msgCh, errCh := claude.Query(ctx, "What is 2 + 2?", nil)
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			if assistantMsg, ok := msg.(*messages.AssistantMessage); ok {
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
4. `mcp/` - SDK MCP server
5. `permissions/` - Permission callbacks
6. `tools/` - Tool filtering
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
