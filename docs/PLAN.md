# Claude Agent SDK Go Implementation Plan

## Executive Summary

This plan outlines the implementation of a comprehensive Go SDK for Claude Agent, based on the Python SDK reference implementation. The Go SDK will provide idiomatic Go interfaces for interacting with Claude Code CLI, supporting both simple one-shot queries and complex bidirectional streaming conversations.

## Architecture Overview

### Core Design Principles

1. **Idiomatic Go**: Use Go conventions (interfaces, channels, contexts, errors)
2. **Type Safety**: Leverage Go's strong typing with generics where appropriate
3. **Concurrency**: Use goroutines and channels for async operations
4. **Error Handling**: Explicit error returns following Go best practices
5. **Context Support**: Full context.Context integration for cancellation and timeouts
6. **Zero Dependencies**: Minimize external dependencies where possible

### Package Structure (Hexagonal Architecture)

Following hexagonal architecture principles (ports and adapters), the SDK separates the core domain from external dependencies. Package names describe what they **provide** (functionality/context), not what they **contain** (generic types).

```
claude-agent-sdk-go/
├── cmd/
│   └── examples/           # Example binaries
│       ├── quickstart/
│       ├── streaming/
│       ├── hooks/
│       └── mcp/
├── pkg/
│   └── claude/
│       # Core Domain (business logic - does not depend on adapters)
│       ├── querying/       # Query execution domain service
│       │   └── service.go
│       ├── streaming/      # Streaming conversation domain service
│       │   └── service.go
│       ├── hooking/        # Hook execution domain logic
│       │   └── service.go
│       ├── permissions/    # Permission handling domain logic
│       │   └── service.go
│       │
│       # Domain Models (shared types used by domain)
│       ├── messages/       # Message type definitions
│       │   └── messages.go
│       ├── options/        # Configuration models
│       │   └── options.go
│       │
│       # Input/Output Ports (interfaces defined by domain)
│       ├── ports/
│       │   ├── transport.go    # Transport port (interface)
│       │   ├── protocol.go     # Protocol port (interface)
│       │   └── storage.go      # Storage port (interface)
│       │
│       # Adapters (implementations of ports - depend on domain)
│       ├── adapters/
│       │   ├── cli/            # CLI subprocess transport adapter
│       │   │   └── transport.go
│       │   ├── jsonrpc/        # Control protocol adapter
│       │   │   └── protocol.go
│       │   └── mcp/            # MCP server adapter
│       │       └── server.go
│       │
│       # Public API (facade over domain services)
│       ├── client.go       # Client (interactive conversations)
│       ├── query.go        # Query() function (one-shot)
│       ├── errors.go       # Error types
│       │
│       └── internal/       # Internal unexported utilities
│           └── parse/
│               └── parser.go
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
└── .golangci.yaml
```

**Hexagonal Architecture Key Principles:**

1. **Core Domain Independence**: Domain packages (`querying`, `streaming`, `hooking`, `permissions`) contain business logic and do NOT import adapters
2. **Ports Define Contracts**: The `ports/` package contains interfaces defined by the domain's needs
3. **Adapters Implement Ports**: Adapters in `adapters/` implement the port interfaces and handle external concerns
4. **Dependency Direction**: Always flows inward - adapters depend on domain, never the reverse
5. **Context-Based Naming**: Packages named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)

## Phase 1: Core Domain & Ports

### 1.1 Domain Models (messages/, options/)

**Priority**: Critical

Define core domain models that are free from infrastructure concerns:

**messages/messages.go - Message Types:**

```go
// Message types
type Message interface {
    message()
}

type UserMessage struct {
    Content          MessageContent
    ParentToolUseID  *string
}

type AssistantMessage struct {
    Content         []ContentBlock
    Model           string
    ParentToolUseID *string
}

type SystemMessage struct {
    Subtype string
    Data    map[string]any
}

type ResultMessage struct {
    Subtype       string
    DurationMs    int
    DurationAPIMs int
    IsError       bool
    NumTurns      int
    SessionID     string
    TotalCostUSD  *float64
    Usage         map[string]any
    Result        *string
}

type StreamEvent struct {
    UUID            string
    SessionID       string
    Event           map[string]any
    ParentToolUseID *string
}

// Content blocks
type ContentBlock interface {
    contentBlock()
}

type TextBlock struct {
    Text string
}

type ThinkingBlock struct {
    Thinking  string
    Signature string
}

type ToolUseBlock struct {
    ID    string
    Name  string
    Input map[string]any
}

type ToolResultBlock struct {
    ToolUseID string
    Content   any // string or []map[string]any
    IsError   *bool
}

// Message content can be string or []ContentBlock
type MessageContent interface {
    messageContent()
}

type StringContent string
type BlockListContent []ContentBlock

func (StringContent) messageContent()    {}
func (BlockListContent) messageContent() {}
```

**options/options.go - Configuration Models:**

```go
package options
type PermissionMode string

const (
    PermissionModeDefault          PermissionMode = "default"
    PermissionModeAcceptEdits      PermissionMode = "acceptEdits"
    PermissionModePlan             PermissionMode = "plan"
    PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

type AgentOptions struct {
    AllowedTools             []string
    SystemPrompt             SystemPromptConfig
    MCPServers               map[string]MCPServerConfig
    PermissionMode           *PermissionMode
    ContinueConversation     bool
    Resume                   *string
    MaxTurns                 *int
    DisallowedTools          []string
    Model                    *string
    PermissionPromptToolName *string
    Cwd                      *string
    Settings                 *string
    AddDirs                  []string
    Env                      map[string]string
    ExtraArgs                map[string]*string
    MaxBufferSize            *int
    StderrCallback           func(string)

    // Callbacks
    CanUseTool               CanUseToolFunc

    // Hooks
    Hooks                    map[HookEvent][]HookMatcher

    // Advanced features
    IncludePartialMessages   bool
    ForkSession              bool
    Agents                   map[string]AgentDefinition
    SettingSources           []SettingSource
    User                     *string
}

type SystemPromptConfig interface {
    systemPromptConfig()
}

type StringSystemPrompt string
type PresetSystemPrompt struct {
    Type   string
    Preset string
    Append *string
}

func (StringSystemPrompt) systemPromptConfig()  {}
func (PresetSystemPrompt) systemPromptConfig() {}

type MCPServerConfig interface {
    mcpServerConfig()
}

type StdioServerConfig struct {
    Type    *string // "stdio" or nil for backward compat
    Command string
    Args    []string
    Env     map[string]string
}

type SSEServerConfig struct {
    Type    string // "sse"
    URL     string
    Headers map[string]string
}

type HTTPServerConfig struct {
    Type    string // "http"
    URL     string
    Headers map[string]string
}

type SDKServerConfig struct {
    Type     string // "sdk"
    Name     string
    Instance MCPServer
}
```

### 1.2 Ports (Interfaces)

**Priority**: Critical

Define port interfaces that the domain needs. These are defined BY the domain, not by external systems.

**ports/transport.go - Transport Port:**

```go
package ports

import "context"

// Transport defines what the domain needs from a transport layer
type Transport interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, data string) error
    ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)
    EndInput() error
    Close() error
    IsReady() bool
}
```

**ports/protocol.go - Protocol Port:**

```go
package ports

import "context"

// ProtocolHandler defines what the domain needs for control protocol
type ProtocolHandler interface {
    Initialize(ctx context.Context, config any) (map[string]any, error)
    SendRequest(ctx context.Context, req map[string]any) (map[string]any, error)
    HandleRequest(ctx context.Context, req map[string]any) (map[string]any, error)
}
```

### 1.3 Error Types (errors.go)

**Priority**: Critical

```go
package claude
var (
    ErrNotConnected      = errors.New("claude: not connected")
    ErrCLINotFound      = errors.New("claude: CLI not found")
    ErrCLIConnection    = errors.New("claude: connection failed")
    ErrProcessFailed    = errors.New("claude: process failed")
    ErrJSONDecode       = errors.New("claude: JSON decode failed")
    ErrMessageParse     = errors.New("claude: message parse failed")
    ErrControlTimeout   = errors.New("claude: control request timeout")
    ErrInvalidInput     = errors.New("claude: invalid input")
)

type CLINotFoundError struct {
    Path string
}

func (e *CLINotFoundError) Error() string {
    return fmt.Sprintf("Claude Code not found: %s", e.Path)
}

type ProcessError struct {
    ExitCode int
    Stderr   string
}

func (e *ProcessError) Error() string {
    return fmt.Sprintf("process failed with exit code %d: %s", e.ExitCode, e.Stderr)
}

type JSONDecodeError struct {
    Line string
    Err  error
}

func (e *JSONDecodeError) Error() string {
    return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

func (e *JSONDecodeError) Unwrap() error {
    return e.Err
}
```

## Phase 2: Domain Services

### 2.1 Querying Service (querying/service.go)

**Priority**: Critical

The querying service encapsulates the domain logic for executing one-shot queries.

```go
package querying

import (
    "context"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/ports"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/messages"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/options"
)

// Service handles query execution
type Service struct {
    transport ports.Transport
    protocol  ports.ProtocolHandler
}

func NewService(transport ports.Transport, protocol ports.ProtocolHandler) *Service {
    return &Service{
        transport: transport,
        protocol:  protocol,
    }
}

func (s *Service) Execute(ctx context.Context, prompt string, opts *options.AgentOptions) (<-chan messages.Message, <-chan error) {
    // Domain logic for query execution
}
```

### 2.2 Streaming Service (streaming/service.go)

**Priority**: Critical

The streaming service handles bidirectional streaming conversations.

```go
package streaming

import (
    "context"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/ports"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/messages"
)

// Service handles streaming conversations
type Service struct {
    transport ports.Transport
    protocol  ports.ProtocolHandler
}

func NewService(transport ports.Transport, protocol ports.ProtocolHandler) *Service {
    return &Service{
        transport: transport,
        protocol:  protocol,
    }
}

func (s *Service) Connect(ctx context.Context, prompt any) error {
    // Domain logic for establishing streaming connection
}

func (s *Service) SendMessage(ctx context.Context, msg string) error {
    // Domain logic for sending messages
}

func (s *Service) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
    // Domain logic for receiving messages
}
```

## Phase 3: Adapters (Infrastructure)

### 3.1 CLI Transport Adapter (adapters/cli/transport.go)

**Priority**: Critical

This adapter implements the Transport port using subprocess CLI.

```go
package cli

import (
    "bufio"
    "context"
    "encoding/json"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/ports"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/options"
    "os/exec"
    "sync"
)

// Adapter implements ports.Transport using CLI subprocess
type Adapter struct {
    options       *options.AgentOptions
    cliPath       string
    cmd           *exec.Cmd
    stdin         io.WriteCloser
    stdout        io.ReadCloser
    ready         bool
    mu            sync.RWMutex
}

// Verify interface compliance at compile time
var _ ports.Transport = (*Adapter)(nil)

func NewAdapter(opts *options.AgentOptions) *Adapter {
    return &Adapter{
        options: opts,
    }
}

func (a *Adapter) Connect(ctx context.Context) error {
    // Implementation: Start CLI subprocess
}

func (a *Adapter) Write(ctx context.Context, data string) error {
    // Implementation: Write to stdin
}

func (a *Adapter) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
    // Implementation: Read from stdout
}

func (a *Adapter) EndInput() error {
    // Implementation: Close stdin
}

func (a *Adapter) Close() error {
    // Implementation: Cleanup subprocess
}

func (a *Adapter) IsReady() bool {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.ready
}
```

### 3.2 JSON-RPC Protocol Adapter (adapters/jsonrpc/protocol.go)

**Priority**: High

```go
package jsonrpc

import (
    "context"
    "encoding/json"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/ports"
    "sync"
)

// Adapter implements ports.ProtocolHandler using JSON-RPC
type Adapter struct {
    transport      ports.Transport
    pendingReqs    map[string]chan result
    requestCounter int
    mu             sync.Mutex
}

// Verify interface compliance
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
    data map[string]any
    err  error
}

func NewAdapter(transport ports.Transport) *Adapter {
    return &Adapter{
        transport:   transport,
        pendingReqs: make(map[string]chan result),
    }
}

func (a *Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
    // Implementation
}

func (a *Adapter) SendRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
    // Implementation
}

func (a *Adapter) HandleRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
    // Implementation
}
```

### 3.3 Message Parser (internal/parse/parser.go)

**Priority**: High

Internal utility for parsing messages (not exposed as port).

```go
package parse

import (
    "fmt"
)

func ParseMessage(data map[string]any) (Message, error) {
    msgType, ok := data["type"].(string)
    if !ok {
        return nil, fmt.Errorf("message missing type field")
    }

    switch msgType {
    case "user":
        return parseUserMessage(data)
    case "assistant":
        return parseAssistantMessage(data)
    case "system":
        return parseSystemMessage(data)
    case "result":
        return parseResultMessage(data)
    case "stream_event":
        return parseStreamEvent(data)
    default:
        return nil, fmt.Errorf("unknown message type: %s", msgType)
    }
}

func parseUserMessage(data map[string]any) (*UserMessage, error) {
    // Implementation
}

func parseAssistantMessage(data map[string]any) (*AssistantMessage, error) {
    // Parse content blocks
    msg, _ := data["message"].(map[string]any)
    contentArray, _ := msg["content"].([]any)

    var blocks []ContentBlock
    for _, item := range contentArray {
        block, _ := item.(map[string]any)
        blockType, _ := block["type"].(string)

        switch blockType {
        case "text":
            text, _ := block["text"].(string)
            blocks = append(blocks, TextBlock{Text: text})
        case "thinking":
            thinking, _ := block["thinking"].(string)
            signature, _ := block["signature"].(string)
            blocks = append(blocks, ThinkingBlock{
                Thinking:  thinking,
                Signature: signature,
            })
        case "tool_use":
            id, _ := block["id"].(string)
            name, _ := block["name"].(string)
            input, _ := block["input"].(map[string]any)
            blocks = append(blocks, ToolUseBlock{
                ID:    id,
                Name:  name,
                Input: input,
            })
        // ... other block types
        }
    }

    model, _ := msg["model"].(string)
    parentToolUseID := getStringPtr(data, "parent_tool_use_id")

    return &AssistantMessage{
        Content:         blocks,
        Model:           model,
        ParentToolUseID: parentToolUseID,
    }, nil
}

// Helper functions
func getStringPtr(data map[string]any, key string) *string {
    if val, ok := data[key].(string); ok {
        return &val
    }
    return nil
}
```

## Phase 4: Public API (Facade)

The public API acts as a facade over the domain services, hiding the complexity of ports and adapters.

### 4.1 Query Function (query.go)

**Priority**: Critical

```go
package claude

import (
    "context"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/querying"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/adapters/cli"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/adapters/jsonrpc"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/messages"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/options"
)

// Query performs a one-shot query to Claude
// This is the main entry point that wires up domain services with adapters
func Query(ctx context.Context, prompt string, opts *options.AgentOptions) (<-chan messages.Message, <-chan error) {
    if opts == nil {
        opts = &options.AgentOptions{}
    }

    // Wire up adapters (infrastructure layer)
    transport := cli.NewAdapter(opts)
    protocol := jsonrpc.NewAdapter(transport)

    // Create domain service
    queryService := querying.NewService(transport, protocol)

    // Execute domain logic
    return queryService.Execute(ctx, prompt, opts)
}
```

### 4.2 Client (client.go)

**Priority**: Critical

```go
package claude

import (
    "context"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/streaming"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/adapters/cli"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/adapters/jsonrpc"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/messages"
    "github.com/yourorg/claude-agent-sdk-go/pkg/claude/options"
    "sync"
)

// Client provides bidirectional, interactive conversations with Claude
// It's a facade that wires domain services with adapters
type Client struct {
    opts            *options.AgentOptions
    streamingService *streaming.Service
    mu              sync.Mutex
}

// NewClient creates a new Claude client
func NewClient(opts *options.AgentOptions) *Client {
    if opts == nil {
        opts = &options.AgentOptions{}
    }
    return &Client{
        opts: opts,
    }
}

// Connect establishes connection to Claude
func (c *Client) Connect(ctx context.Context, prompt any) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Wire up adapters (infrastructure)
    transport := cli.NewAdapter(c.opts)
    protocol := jsonrpc.NewAdapter(transport)

    // Create domain service
    c.streamingService = streaming.NewService(transport, protocol)

    // Execute domain logic
    return c.streamingService.Connect(ctx, prompt)
}

// SendMessage sends a message to Claude
func (c *Client) SendMessage(ctx context.Context, msg string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.streamingService == nil {
        return ErrNotConnected
    }

    return c.streamingService.SendMessage(ctx, msg)
}

// ReceiveMessages returns a channel of messages from Claude
func (c *Client) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
    if c.streamingService == nil {
        errCh := make(chan error, 1)
        errCh <- ErrNotConnected
        close(errCh)
        return nil, errCh
    }

    return c.streamingService.ReceiveMessages(ctx)
}

// Close disconnects from Claude
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.streamingService == nil {
        return nil
    }

    // Domain service handles cleanup
    return c.streamingService.Close()
}
```

## Phase 5: Advanced Features

### 5.1 Hooks Support (hooks.go)

**Priority**: Medium

```go
package claude

type HookEvent string

const (
    HookEventPreToolUse       HookEvent = "PreToolUse"
    HookEventPostToolUse      HookEvent = "PostToolUse"
    HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
    HookEventStop             HookEvent = "Stop"
    HookEventSubagentStop     HookEvent = "SubagentStop"
    HookEventPreCompact       HookEvent = "PreCompact"
)

type HookContext struct {
    // Future: signal support for cancellation
}

type HookCallbackFunc func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error)

type HookMatcher struct {
    Matcher string
    Hooks   []HookCallbackFunc
}

type HookJSONOutput struct {
    Decision           *string        `json:"decision,omitempty"`           // "block"
    SystemMessage      *string        `json:"systemMessage,omitempty"`
    HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// Example hook implementation
func BlockBashPatternHook(patterns []string) HookCallbackFunc {
    return func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
        toolName, _ := input["tool_name"].(string)
        if toolName != "Bash" {
            return map[string]any{}, nil
        }

        toolInput, _ := input["tool_input"].(map[string]any)
        command, _ := toolInput["command"].(string)

        for _, pattern := range patterns {
            if strings.Contains(command, pattern) {
                return map[string]any{
                    "hookSpecificOutput": map[string]any{
                        "hookEventName":            "PreToolUse",
                        "permissionDecision":       "deny",
                        "permissionDecisionReason": fmt.Sprintf("Command contains forbidden pattern: %s", pattern),
                    },
                }, nil
            }
        }

        return map[string]any{}, nil
    }
}
```

### 5.2 MCP Server Support (mcp.go)

**Priority**: Medium

```go
package claude

import (
    "context"
    "fmt"
)

// MCPServer interface for in-process MCP servers
type MCPServer interface {
    Name() string
    Version() string
    ListTools(ctx context.Context) ([]MCPTool, error)
    CallTool(ctx context.Context, name string, args map[string]any) (MCPToolResult, error)
}

type MCPTool struct {
    Name        string
    Description string
    InputSchema map[string]any
}

type MCPToolResult struct {
    Content []MCPContent
    IsError bool
}

type MCPContent struct {
    Type     string
    Text     string
    Data     string
    MimeType string
}

// Tool decorator and server builder
type SDKMCPTool struct {
    Name        string
    Description string
    InputSchema any
    Handler     func(context.Context, map[string]any) (map[string]any, error)
}

func Tool(name, description string, inputSchema any, handler func(context.Context, map[string]any) (map[string]any, error)) *SDKMCPTool {
    return &SDKMCPTool{
        Name:        name,
        Description: description,
        InputSchema: inputSchema,
        Handler:     handler,
    }
}

type sdkMCPServer struct {
    name    string
    version string
    tools   map[string]*SDKMCPTool
}

func CreateSDKMCPServer(name, version string, tools []*SDKMCPTool) MCPServerConfig {
    toolMap := make(map[string]*SDKMCPTool)
    for _, tool := range tools {
        toolMap[tool.Name] = tool
    }

    server := &sdkMCPServer{
        name:    name,
        version: version,
        tools:   toolMap,
    }

    return SDKServerConfig{
        Type:     "sdk",
        Name:     name,
        Instance: server,
    }
}

func (s *sdkMCPServer) Name() string {
    return s.name
}

func (s *sdkMCPServer) Version() string {
    return s.version
}

func (s *sdkMCPServer) ListTools(ctx context.Context) ([]MCPTool, error) {
    var tools []MCPTool
    for _, tool := range s.tools {
        schema := convertInputSchema(tool.InputSchema)
        tools = append(tools, MCPTool{
            Name:        tool.Name,
            Description: tool.Description,
            InputSchema: schema,
        })
    }
    return tools, nil
}

func (s *sdkMCPServer) CallTool(ctx context.Context, name string, args map[string]any) (MCPToolResult, error) {
    tool, exists := s.tools[name]
    if !exists {
        return MCPToolResult{}, fmt.Errorf("tool not found: %s", name)
    }

    result, err := tool.Handler(ctx, args)
    if err != nil {
        return MCPToolResult{
            Content: []MCPContent{{Type: "text", Text: err.Error()}},
            IsError: true,
        }, nil
    }

    // Convert result to MCP format
    var content []MCPContent
    if contentList, ok := result["content"].([]any); ok {
        for _, item := range contentList {
            if itemMap, ok := item.(map[string]any); ok {
                itemType, _ := itemMap["type"].(string)
                text, _ := itemMap["text"].(string)
                content = append(content, MCPContent{
                    Type: itemType,
                    Text: text,
                })
            }
        }
    }

    isError, _ := result["is_error"].(bool)

    return MCPToolResult{
        Content: content,
        IsError: isError,
    }, nil
}

func convertInputSchema(schema any) map[string]any {
    switch s := schema.(type) {
    case map[string]any:
        // Check if already JSON schema
        if _, hasType := s["type"]; hasType {
            return s
        }
        // Convert simple map to JSON schema
        properties := make(map[string]any)
        required := []string{}
        for name, typ := range s {
            required = append(required, name)
            switch typ {
            case "string":
                properties[name] = map[string]any{"type": "string"}
            case "int", "integer":
                properties[name] = map[string]any{"type": "integer"}
            case "float", "number":
                properties[name] = map[string]any{"type": "number"}
            case "bool", "boolean":
                properties[name] = map[string]any{"type": "boolean"}
            default:
                properties[name] = map[string]any{"type": "string"}
            }
        }
        return map[string]any{
            "type":       "object",
            "properties": properties,
            "required":   required,
        }
    default:
        return map[string]any{"type": "object", "properties": map[string]any{}}
    }
}
```

### 5.3 Permission Callbacks (permissions.go)

**Priority**: Medium

```go
package claude

import (
    "context"
)

type ToolPermissionContext struct {
    Suggestions []PermissionUpdate
}

type PermissionResult interface {
    permissionResult()
}

type PermissionResultAllow struct {
    UpdatedInput       map[string]any
    UpdatedPermissions []PermissionUpdate
}

type PermissionResultDeny struct {
    Message   string
    Interrupt bool
}

func (PermissionResultAllow) permissionResult() {}
func (PermissionResultDeny) permissionResult()  {}

type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)

type PermissionUpdate struct {
    Type        string
    Rules       []PermissionRuleValue
    Behavior    *PermissionBehavior
    Mode        *PermissionMode
    Directories []string
    Destination *PermissionUpdateDestination
}

type PermissionRuleValue struct {
    ToolName    string
    RuleContent *string
}

type PermissionBehavior string

const (
    PermissionBehaviorAllow PermissionBehavior = "allow"
    PermissionBehaviorDeny  PermissionBehavior = "deny"
    PermissionBehaviorAsk   PermissionBehavior = "ask"
)

type PermissionUpdateDestination string

const (
    PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
    PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
    PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
    PermissionDestinationSession         PermissionUpdateDestination = "session"
)
```

## Phase 6: Testing & Documentation

### 6.1 Unit Tests

**Priority**: Critical

- Test all message parsing
- Test transport layer
- Test control protocol
- Test error handling
- Mock Claude CLI for testing

### 6.2 Integration Tests

**Priority**: High

- Test against actual Claude CLI
- Test streaming mode
- Test hooks
- Test MCP servers

### 6.3 Examples

**Priority**: High

Create comprehensive examples for quick start, streaming, hooks, and MCP servers.

### 6.4 Documentation

**Priority**: High

- Comprehensive README.md
- API documentation with godoc
- Migration guide from Python SDK
- Architecture documentation

## Phase 7: Publishing & CI/CD

### 7.1 Go Module Setup

**Priority**: Critical

### 7.2 CI/CD Pipeline

**Priority**: High

### 7.3 Release Process

**Priority**: Medium

## Implementation Phases

### Phase 1: Foundation
- Core Foundation (types, options, errors)
- Transport Layer (interface, subprocess)
- Start Protocol Layer

### Phase 2: Core Implementation
- Complete Protocol Layer (control protocol, message parser)
- Public API (query function, client)

### Phase 3: Advanced Features
- Advanced Features (hooks, MCP, permissions)
- Start Testing

### Phase 4: Polish & Release
- Complete Testing & Documentation
- Publishing & CI/CD

## Key Design Decisions

### Hexagonal Architecture Principles

1. **Domain Independence**: Core domain packages (`querying`, `streaming`, `hooking`, `permissions`) never import adapters
2. **Ports Define Contracts**: Interfaces in `ports/` package are defined by domain needs, not external systems
3. **Adapters Implement Ports**: All infrastructure code in `adapters/` implements port interfaces
4. **Dependency Direction**: Always flows inward (adapters → domain), never outward (domain → adapters)
5. **Package Naming**: Named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)

### Go Idioms

6. **Channels vs Iterators**: Use channels for async message streaming (idiomatic Go)
7. **Context Integration**: Full context.Context support throughout
8. **Error Handling**: Return errors explicitly, use error wrapping
9. **Interface Compliance**: Use `var _ ports.Transport = (*Adapter)(nil)` pattern to verify at compile time
10. **Async Model**: Goroutines + channels (Go's native async)
11. **JSON Handling**: Use encoding/json with struct tags
12. **Testing Strategy**: Table-driven tests, interface mocks, integration tests

### Architectural Benefits

- **Testability**: Domain logic testable without infrastructure dependencies
- **Flexibility**: Easy to swap adapters (e.g., different transport mechanisms)
- **Clarity**: Clear separation between business logic and technical details
- **Maintainability**: Changes to infrastructure don't affect domain
- **Discoverability**: Package names describe purpose at a glance

## Success Criteria

1. Functional parity with Python SDK
2. Clean, idiomatic Go API
3. Efficient resource usage
4. Comprehensive documentation
5. >80% test coverage
6. Automated CI/CD
7. Easy to use and well-documented

## Conclusion

This plan provides a comprehensive roadmap for implementing a production-ready Claude Agent SDK for Go. The implementation follows Go best practices while maintaining functional parity with the Python SDK.
