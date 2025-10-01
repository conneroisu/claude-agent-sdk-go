# Claude Agent SDK Go Implementation Plan
## Executive Summary
This plan outlines the implementation of a comprehensive Go SDK for Claude Agent, based on the Python SDK reference implementation (REQUIRED that you see ./claude-agent-sdk-python)
The Go SDK will provide idiomatic Go interfaces for interacting with Claude Code CLI, supporting both simple one-shot queries and complex bidirectional streaming conversations.
## Architecture Overview
### Core Design Principles
1. Idiomatic Go: Use Go conventions (interfaces, channels, contexts, errors)
2. Type Safety: Leverage Go's strong typing with generics where appropriate
3. Concurrency: Use goroutines and channels for async operations
4. Error Handling: Explicit error returns following Go best practices
5. Context Support: Full context.Context integration for cancellation and timeouts
6. Zero Dependencies: Minimize external dependencies where possible
7. Hexagonal Architecture: Strict separation between domain logic and infrastructure
### Hexagonal Architecture (Ports and Adapters)
This SDK follows hexagonal architecture principles, also known as ports and adapters pattern. This architectural style isolates the core business logic (domain) from external concerns (infrastructure) by defining clear boundaries and dependency rules.
#### Key Concepts
```
         ┌─────────────────────────────────────────────┐
         │                                             │
         │      External World (Infrastructure)        │
         │                                             │
         │   ┌──────────┐              ┌──────────┐   │
         │   │   HTTP   │              │   CLI    │   │
         │   │ Handlers │              │ Adapter  │   │
         │   └────┬─────┘              └─────┬────┘   │
         │        │                          │        │
         │        │   ┌───────────────┐      │        │
         │        └──►│     Ports     │◄─────┘        │
         │            │  (Interfaces) │               │
         │            └───────┬───────┘               │
         │                    │                       │
         │       ┌────────────▼────────────┐          │
         │       │                         │          │
         │       │    Core Domain          │          │
         │       │  (Business Logic)       │          │
         │       │                         │          │
         │       │  - querying/            │          │
         │       │  - streaming/           │          │
         │       │  - hooking/             │          │
         │       │  - permissions/         │          │
         │       │                         │          │
         │       └─────────────────────────┘          │
         │                                             │
         └─────────────────────────────────────────────┘
```
Four Key Principles:
1. Domain Independence: Core domain never imports adapters or infrastructure code
2. Ports Define Contracts: Interfaces defined by domain needs, not external systems
3. Adapters Implement Ports: Infrastructure code implements domain-defined interfaces
4. Dependency Direction: Always flows inward (adapters → domain), never outward
Why Hexagonal Architecture?
- Testability: Test domain logic without databases, HTTP servers, or external services
- Flexibility: Swap implementations (e.g., different storage, transport mechanisms) without changing business logic
- Clarity: Clear separation between "what" (domain) and "how" (infrastructure)
- Maintainability: Infrastructure changes don't affect domain; domain changes don't ripple to all adapters
### Package Structure (Hexagonal Architecture)
Following hexagonal architecture principles (ports and adapters), the SDK separates the core domain from external dependencies. Package names describe what they provide (functionality/context), not what they contain (generic types).
```
claude-agent-sdk-go/
├── cmd/                        # ═══ BINARIES (Entry Points) ═══
│   └── examples/               # Example applications
│       ├── quickstart/
│       ├── streaming/
│       ├── hooks/
│       └── mcp/
│
├── pkg/claude/
│   # ═══════════════════════════════════════════════════════
│   # LAYER 1: CORE DOMAIN (Business Logic)
│   # - Never imports from adapters/
│   # - Only imports from ports/ (interfaces it defines)
│   # - Pure business logic, no infrastructure concerns
│   # ═══════════════════════════════════════════════════════
│   ├── querying/               # Domain service: "Execute one-shot queries"
│   │   └── service.go
│   ├── streaming/              # Domain service: "Manage streaming conversations"
│   │   └── service.go
│   ├── hooking/                # Domain service: "Execute lifecycle hooks"
│   │   └── service.go
│   ├── permissions/            # Domain service: "Check tool permissions"
│   │   └── service.go
│   │
│   # ═══════════════════════════════════════════════════════
│   # LAYER 1B: DOMAIN MODELS
│   # - Shared types used across domain
│   # - No infrastructure dependencies
│   # ═══════════════════════════════════════════════════════
│   ├── messages/               # Domain models: Message types
│   │   └── messages.go
│   ├── options/                # Domain models: Configuration
│   │   ├── domain.go           # Pure domain options (PermissionMode, etc.)
│   │   ├── transport.go        # Transport configuration
│   │   └── mcp.go              # MCP server configuration
│   │
│   # ═══════════════════════════════════════════════════════
│   # LAYER 2: PORTS (Domain-Defined Interfaces)
│   # - Interfaces defined BY domain needs
│   # - NOT defined by external systems
│   # - This is the "contract" layer
│   # ═══════════════════════════════════════════════════════
│   ├── ports/
│   │   ├── transport.go        # What domain needs from transport
│   │   ├── protocol.go         # What domain needs from control protocol
│   │   ├── parser.go           # What domain needs from message parsing
│   │   └── mcp.go              # What domain needs from MCP servers
│   │
│   # ═══════════════════════════════════════════════════════
│   # LAYER 3: ADAPTERS (Infrastructure Implementations)
│   # - Implements port interfaces
│   # - Handles external concerns (CLI, JSON-RPC, parsing)
│   # - Can import from domain and ports
│   # - Domain NEVER imports from here
│   # ═══════════════════════════════════════════════════════
│   ├── adapters/
│   │   ├── cli/                # Adapter: CLI subprocess transport
│   │   │   └── transport.go    # Implements ports.Transport
│   │   ├── jsonrpc/            # Adapter: Control protocol handler
│   │   │   └── protocol.go     # Implements ports.ProtocolHandler
│   │   ├── parse/              # Adapter: Message parser
│   │   │   └── parser.go       # Implements ports.MessageParser
│   │   └── mcp/                # Adapter: MCP server implementation
│   │       └── server.go       # Implements ports.MCPServer
│   │
│   # ═══════════════════════════════════════════════════════
│   # LAYER 4: PUBLIC API (Facade)
│   # - Wires domain services with adapters
│   # - Entry point for SDK users
│   # - Hides complexity of ports/adapters from users
│   # ═══════════════════════════════════════════════════════
│   ├── client.go               # Client for interactive conversations
│   ├── query.go                # Query() for one-shot requests
│   └── errors.go               # Public error types
│
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
└── .golangci.yaml
```
Hexagonal Architecture Key Principles:
1. Core Domain Independence: Domain packages (`querying`, `streaming`, `hooking`, `permissions`) contain business logic and do NOT import adapters
2. Ports Define Contracts: The `ports/` package contains interfaces defined by the domain's needs
3. Adapters Implement Ports: Adapters in `adapters/` implement the port interfaces and handle external concerns
4. Dependency Direction: Always flows inward - adapters depend on domain, never the reverse
5. Context-Based Naming: Packages named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)
## Phase 1: Core Domain & Ports
### 1.1 Domain Models (messages/, options/)
Priority: Critical
Define core domain models that are free from infrastructure concerns:
messages/messages.go - Message Types:
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
options/domain.go - Pure Domain Configuration:
```go
package options
// PermissionMode defines how permissions are handled
// This is a DOMAIN concept - it affects business logic
type PermissionMode string
const (
    PermissionModeDefault          PermissionMode = "default"
    PermissionModeAcceptEdits      PermissionMode = "acceptEdits"
    PermissionModePlan             PermissionMode = "plan"
    PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)
// SettingSource specifies where settings come from
type SettingSource string
const (
    SettingSourceUser    SettingSource = "user"
    SettingSourceProject SettingSource = "project"
    SettingSourceLocal   SettingSource = "local"
)
// AgentDefinition defines a subagent configuration
// This is domain configuration - defines behavior of agents
type AgentDefinition struct {
    Name          string
    Description   string
    SystemPrompt  *string
    AllowedTools  []string
    Model         *string
}
// SystemPromptConfig is configuration for system prompts
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
```
options/transport.go - Transport/Infrastructure Configuration:
```go
package options
// AgentOptions configures the Claude agent
// This combines domain and infrastructure configuration
type AgentOptions struct {
    // Domain settings (affect business logic)
    AllowedTools             []string
    DisallowedTools          []string
    Model                    *string
    MaxTurns                 *int
    SystemPrompt             SystemPromptConfig
    PermissionMode           *PermissionMode
    PermissionPromptToolName *string
    Agents                   map[string]AgentDefinition
    // Session management (domain concern)
    ContinueConversation     bool
    Resume                   *string
    ForkSession              bool
    IncludePartialMessages   bool
    // Infrastructure settings (how to connect/execute)
    Cwd                      *string
    Settings                 *string
    AddDirs                  []string
    Env                      map[string]string
    User                     *string
    SettingSources           []SettingSource
    MaxBufferSize            *int
    StderrCallback           func(string)
    ExtraArgs                map[string]*string
    // MCP server configuration (infrastructure)
    MCPServers               map[string]MCPServerConfig
    // Internal flags (set by domain services, not by users)
    _isStreaming             bool  // Internal: true for Client, false for Query
}
```
options/mcp.go - MCP Server Configuration:
```go
package options
// MCPServerConfig is configuration for MCP servers (not runtime instances)
// These are infrastructure configurations for connecting to MCP servers
type MCPServerConfig interface {
    mcpServerConfig()
}
// StdioServerConfig configures an MCP server using stdio transport
type StdioServerConfig struct {
    Type    string // "stdio"
    Command string
    Args    []string
    Env     map[string]string
}
func (StdioServerConfig) mcpServerConfig() {}
// SSEServerConfig configures an MCP server using Server-Sent Events
type SSEServerConfig struct {
    Type    string // "sse"
    URL     string
    Headers map[string]string
}
func (SSEServerConfig) mcpServerConfig() {}
// HTTPServerConfig configures an MCP server using HTTP transport
type HTTPServerConfig struct {
    Type    string // "http"
    URL     string
    Headers map[string]string
}
func (HTTPServerConfig) mcpServerConfig() {}
// SDKServerConfig is a marker for SDK-managed MCP servers
// The actual server instance is managed separately by the MCP adapter
// This ONLY contains configuration, NOT the server instance itself
type SDKServerConfig struct {
    Type string // "sdk"
    Name string
    // Note: Instance is NOT stored here to avoid circular dependencies
    // The MCP adapter will manage server instances separately
}
func (SDKServerConfig) mcpServerConfig() {}
```
### 1.2 Ports (Interfaces)
Priority: Critical
Define port interfaces that the domain needs. These are defined BY the domain, not by external systems.
ports/transport.go - Transport Port:
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
ports/protocol.go - Protocol Port:
```go
package ports
import "context"
// ProtocolHandler defines what the domain needs for control protocol
type ProtocolHandler interface {
    // Initialize sends the initialize control request with hooks config
    Initialize(ctx context.Context, config any) (map[string]any, error)
    // SendControlRequest sends a control request and waits for response (60s timeout)
    SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error)
    // HandleControlRequest routes inbound control requests by subtype
    // Subtypes: can_use_tool, hook_callback, mcp_message
    // Dependencies are passed as arguments to avoid circular refs
    HandleControlRequest(ctx context.Context, req map[string]any, perms *permissions.Service, hooks map[string]hooking.HookCallback, mcpServers map[string]MCPServer) (map[string]any, error)
    // StartMessageRouter continuously reads transport and partitions messages
    // Routes control_response, control_request, control_cancel_request separately from SDK messages
    // Dependencies (perms, hooks, mcpServers) are passed by domain service for handling inbound control requests
    StartMessageRouter(ctx context.Context, msgCh chan<- map[string]any, errCh chan<- error,
        perms *permissions.Service, hooks map[string]hooking.HookCallback, mcpServers map[string]MCPServer) error
}
```
ports/parser.go - Message Parser Port:
```go
package ports
import (
    "github.com/conneroisu/claude/pkg/claude/messages"
)
// MessageParser defines what the domain needs from message parsing
// This is a port because the domain needs to convert raw transport messages
// into typed domain messages, but doesn't care HOW that conversion happens
type MessageParser interface {
    Parse(raw map[string]any) (messages.Message, error)
}
```
ports/mcp.go - MCP Server Port:
```go
package ports
import "context"
// MCPServer defines what the domain needs from MCP server implementations
// This is a port because the domain needs to interact with MCP servers
// but doesn't care about their internal implementation
type MCPServer interface {
    Name() string
    Initialize(ctx context.Context, params any) (any, error)
    ListTools(ctx context.Context) ([]MCPTool, error)
    CallTool(ctx context.Context, name string, args map[string]any) (MCPToolResult, error)
    HandleNotification(ctx context.Context, method string, params any) error
}
// MCPTool represents an MCP tool definition
type MCPTool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}
// MCPToolResult represents the result of calling an MCP tool
type MCPToolResult struct {
    Content []map[string]any `json:"content"`
    IsError bool             `json:"isError,omitempty"`
}
```
### 1.3 Error Types (errors.go)
Priority: Critical
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
Priority: Critical
The querying service encapsulates the domain logic for executing one-shot queries.
Key Design Decision: Control protocol state management (pending requests, callback IDs, request counters) is handled by the `jsonrpc` adapter, NOT by domain services. The domain only uses the port interface.
```go
package querying
import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/conneroisu/claude/pkg/claude/ports"
    "github.com/conneroisu/claude/pkg/claude/messages"
    "github.com/conneroisu/claude/pkg/claude/options"
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/permissions"
)
// Service handles query execution
// This is a DOMAIN service - it contains only business logic,
// no infrastructure concerns like protocol state management
type Service struct {
    transport   ports.Transport
    protocol    ports.ProtocolHandler
    parser      ports.MessageParser
    hooks       *hooking.Service
    permissions *permissions.Service
    mcpServers  map[string]ports.MCPServer
}
func NewService(
    transport ports.Transport,
    protocol ports.ProtocolHandler,
    parser ports.MessageParser,
    hooks *hooking.Service,
    perms *permissions.Service,
    mcpServers map[string]ports.MCPServer,
) *Service {
    return &Service{
        transport:   transport,
        protocol:    protocol,
        parser:      parser,
        hooks:       hooks,
        permissions: perms,
        mcpServers:  mcpServers,
    }
}
func (s *Service) Execute(ctx context.Context, prompt string, opts *options.AgentOptions) (<-chan messages.Message, <-chan error) {
    msgCh := make(chan messages.Message)
    errCh := make(chan error, 1)
    go func() {
        defer close(msgCh)
        defer close(errCh)
        // 1. Connect transport
        if err := s.transport.Connect(ctx); err != nil {
            errCh <- fmt.Errorf("transport connect: %w", err)
            return
        }
        // 2. Build hook callbacks map (if hooks exist)
        var hookCallbacks map[string]hooking.HookCallback
        if s.hooks != nil {
            hookCallbacks = make(map[string]hooking.HookCallback)
            hooks := s.hooks.GetHooks()
            for event, matchers := range hooks {
                for _, matcher := range matchers {
                    for i, callback := range matcher.Hooks {
                        // Generate callback ID
                        callbackID := fmt.Sprintf("hook_%s_%d", event, i)
                        hookCallbacks[callbackID] = callback
                    }
                }
            }
        }
        // 3. Start message router (protocol adapter handles control protocol)
        // For one-shot queries, we don't need explicit initialization
        // The protocol adapter will handle any necessary control messages
        routerMsgCh := make(chan map[string]any)
        routerErrCh := make(chan error, 1)
        if err := s.protocol.StartMessageRouter(
            ctx,
            routerMsgCh,
            routerErrCh,
            s.permissions,
            hookCallbacks,
            s.mcpServers,
        ); err != nil {
            errCh <- fmt.Errorf("start message router: %w", err)
            return
        }
        // 4. Send prompt
        promptMsg := map[string]any{
            "type":   "user",
            "prompt": prompt,
        }
        promptBytes, err := json.Marshal(promptMsg)
        if err != nil {
            errCh <- fmt.Errorf("marshal prompt: %w", err)
            return
        }
        if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
            errCh <- fmt.Errorf("write prompt: %w", err)
            return
        }
        // 5. Stream messages
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-routerMsgCh:
                if !ok {
                    return
                }
                // Parse message using parser port
                parsedMsg, err := s.parser.Parse(msg)
                if err != nil {
                    errCh <- fmt.Errorf("parse message: %w", err)
                    return
                }
                msgCh <- parsedMsg
            case err := <-routerErrCh:
                if err != nil {
                    errCh <- err
                    return
                }
            }
        }
    }()
    return msgCh, errCh
}
```
### 2.2 Streaming Service (streaming/service.go)
Priority: Critical
The streaming service handles bidirectional streaming conversations.
Key Design Decision: Like the querying service, control protocol state management is delegated to the protocol adapter. The domain service focuses purely on conversation flow logic.
```go
package streaming
import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/conneroisu/claude/pkg/claude/ports"
    "github.com/conneroisu/claude/pkg/claude/messages"
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/permissions"
)
// Service handles streaming conversations
// This is a DOMAIN service - pure business logic for managing conversations
type Service struct {
    transport   ports.Transport
    protocol    ports.ProtocolHandler
    parser      ports.MessageParser
    hooks       *hooking.Service
    permissions *permissions.Service
    mcpServers  map[string]ports.MCPServer
    // Message routing channels (internal to service)
    msgCh chan map[string]any
    errCh chan error
}
func NewService(
    transport ports.Transport,
    protocol ports.ProtocolHandler,
    parser ports.MessageParser,
    hooks *hooking.Service,
    perms *permissions.Service,
    mcpServers map[string]ports.MCPServer,
) *Service {
    return &Service{
        transport:   transport,
        protocol:    protocol,
        parser:      parser,
        hooks:       hooks,
        permissions: perms,
        mcpServers:  mcpServers,
        msgCh:       make(chan map[string]any),
        errCh:       make(chan error, 1),
    }
}
func (s *Service) Connect(ctx context.Context, prompt any) error {
    // 1. Connect transport
    if err := s.transport.Connect(ctx); err != nil {
        return fmt.Errorf("transport connect: %w", err)
    }
    // 2. Build hook callbacks map
    var hookCallbacks map[string]hooking.HookCallback
    if s.hooks != nil {
        hookCallbacks = make(map[string]hooking.HookCallback)
        hooks := s.hooks.GetHooks()
        for event, matchers := range hooks {
            for _, matcher := range matchers {
                for i, callback := range matcher.Hooks {
                    callbackID := fmt.Sprintf("hook_%s_%d", event, i)
                    hookCallbacks[callbackID] = callback
                }
            }
        }
    }
    // 3. Start message router
    // Protocol adapter handles all control protocol concerns
    if err := s.protocol.StartMessageRouter(
        ctx,
        s.msgCh,
        s.errCh,
        s.permissions,
        hookCallbacks,
        s.mcpServers,
    ); err != nil {
        return fmt.Errorf("start message router: %w", err)
    }
    // 4. Send initial prompt if provided
    if prompt != nil {
        promptMsg := map[string]any{
            "type":   "user",
            "prompt": prompt,
        }
        promptBytes, err := json.Marshal(promptMsg)
        if err != nil {
            return fmt.Errorf("marshal prompt: %w", err)
        }
        if err := s.transport.Write(ctx, string(promptBytes)+"\n"); err != nil {
            return fmt.Errorf("write prompt: %w", err)
        }
    }
    return nil
}
func (s *Service) SendMessage(ctx context.Context, msg string) error {
    // Format message
    userMsg := map[string]any{
        "type":   "user",
        "prompt": msg,
    }
    // Send via transport
    msgBytes, err := json.Marshal(userMsg)
    if err != nil {
        return fmt.Errorf("marshal message: %w", err)
    }
    if err := s.transport.Write(ctx, string(msgBytes)+"\n"); err != nil {
        return fmt.Errorf("write message: %w", err)
    }
    return nil
}
func (s *Service) ReceiveMessages(ctx context.Context) (<-chan messages.Message, <-chan error) {
    msgOutCh := make(chan messages.Message)
    errOutCh := make(chan error, 1)
    go func() {
        defer close(msgOutCh)
        defer close(errOutCh)
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-s.msgCh:
                if !ok {
                    return
                }
                // Parse message using parser port
                parsedMsg, err := s.parser.Parse(msg)
                if err != nil {
                    errOutCh <- fmt.Errorf("parse message: %w", err)
                    return
                }
                msgOutCh <- parsedMsg
            case err := <-s.errCh:
                if err != nil {
                    errOutCh <- err
                    return
                }
            }
        }
    }()
    return msgOutCh, errOutCh
}
func (s *Service) Close() error {
    // Close transport
    if s.transport != nil {
        return s.transport.Close()
    }
    return nil
}
```
### 2.3 Hooking Service (hooking/service.go)
Priority: High
The hooking service manages hook execution and lifecycle.
```go
package hooking
import (
    "context"
)
// HookEvent represents different hook trigger points
type HookEvent string
const (
    HookEventPreToolUse       HookEvent = "PreToolUse"
    HookEventPostToolUse      HookEvent = "PostToolUse"
    HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
    HookEventStop             HookEvent = "Stop"
    HookEventSubagentStop     HookEvent = "SubagentStop"
    HookEventPreCompact       HookEvent = "PreCompact"
)
// HookContext provides context for hook execution
type HookContext struct {
    // Future: signal support for cancellation
}
// HookCallback is a function that handles hook events
type HookCallback func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error)
// HookMatcher defines when a hook should execute
type HookMatcher struct {
    Matcher string          // Pattern to match (e.g., tool name, event type)
    Hooks   []HookCallback  // Callbacks to execute
}
// Service manages hook execution
type Service struct {
    hooks map[HookEvent][]HookMatcher
}
func NewService(hooks map[HookEvent][]HookMatcher) *Service {
    return &Service{
        hooks: hooks,
    }
}
// GetHooks returns the hook configuration
func (s *Service) GetHooks() map[HookEvent][]HookMatcher {
    if s == nil {
        return nil
    }
    return s.hooks
}
// Execute runs hooks for a given event
func (s *Service) Execute(ctx context.Context, event HookEvent, input map[string]any, toolUseID *string) (map[string]any, error) {
    if s == nil || s.hooks == nil {
        return nil, nil
    }
    // 1. Find matching hooks for event
    matchers, exists := s.hooks[event]
    if !exists || len(matchers) == 0 {
        return nil, nil
    }
    // 2. Execute hooks in order and aggregate results
    aggregatedResult := map[string]any{}
    hookCtx := HookContext{}
    for _, matcher := range matchers {
        // Check if matcher applies to this input
        // TODO: Implement pattern matching logic based on matcher.Matcher field
        for _, callback := range matcher.Hooks {
            // 3. Execute hook callback
            result, err := callback(input, toolUseID, hookCtx)
            if err != nil {
                return nil, fmt.Errorf("hook execution failed: %w", err)
            }
            if result == nil {
                continue
            }
            // 4. Handle blocking decisions
            // If hook returns decision="block", stop execution immediately
            if decision, ok := result["decision"].(string); ok && decision == "block" {
                return result, nil
            }
            // Aggregate results (later hooks can override earlier ones)
            for k, v := range result {
                aggregatedResult[k] = v
            }
        }
    }
    return aggregatedResult, nil
}
// Register adds a new hook
func (s *Service) Register(event HookEvent, matcher HookMatcher) {
    if s.hooks == nil {
        s.hooks = make(map[HookEvent][]HookMatcher)
    }
    s.hooks[event] = append(s.hooks[event], matcher)
}
```
### 2.4 Permissions Service (permissions/service.go)
Priority: High
The permissions service handles tool permission checks and updates.
```go
package permissions
import (
    "context"
    "github.com/conneroisu/claude/pkg/claude/options"
)
// PermissionResult represents the outcome of a permission check
type PermissionResult interface {
    permissionResult()
}
// PermissionResultAllow indicates tool use is allowed
type PermissionResultAllow struct {
    UpdatedInput       map[string]any
    UpdatedPermissions []PermissionUpdate
}
func (PermissionResultAllow) permissionResult() {}
// PermissionResultDeny indicates tool use is denied
type PermissionResultDeny struct {
    Message   string
    Interrupt bool
}
func (PermissionResultDeny) permissionResult() {}
// PermissionUpdate represents a permission change
type PermissionUpdate struct {
    Type        string
    Rules       []PermissionRuleValue
    Behavior    *PermissionBehavior
    Mode        *options.PermissionMode
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
// ToolPermissionContext provides context for permission decisions
type ToolPermissionContext struct {
    Suggestions []PermissionUpdate
}
// CanUseToolFunc is a callback for permission checks
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)
// PermissionsConfig holds permission service configuration
type PermissionsConfig struct {
    Mode       options.PermissionMode
    CanUseTool CanUseToolFunc
}
// Service manages tool permissions
type Service struct {
    mode        options.PermissionMode
    canUseTool  CanUseToolFunc
}
func NewService(config *PermissionsConfig) *Service {
    if config == nil {
        return &Service{
            mode: options.PermissionModeAsk,
        }
    }
    return &Service{
        mode:       config.Mode,
        canUseTool: config.CanUseTool,
    }
}
// CheckToolUse verifies if a tool can be used
func (s *Service) CheckToolUse(ctx context.Context, toolName string, input map[string]any) (PermissionResult, error) {
    // 1. Check permission mode
    switch s.mode {
    case options.PermissionModeBypassPermissions:
        // Always allow
        return &PermissionResultAllow{}, nil
    case options.PermissionModeDefault, options.PermissionModeAcceptEdits, options.PermissionModePlan:
        // 2. Call canUseTool callback if set
        if s.canUseTool != nil {
            permCtx := ToolPermissionContext{
                // TODO: Extract suggestions from control request if available
                Suggestions: []PermissionUpdate{},
            }
            result, err := s.canUseTool(ctx, toolName, input, permCtx)
            if err != nil {
                return nil, fmt.Errorf("permission callback failed: %w", err)
            }
            return result, nil
        }
        // 3. Apply default behavior (ask user via CLI)
        // In default mode without callback, we allow but this should be handled by CLI
        return &PermissionResultAllow{}, nil
    default:
        // Unknown mode - deny for safety
        return &PermissionResultDeny{
            Message:   fmt.Sprintf("unknown permission mode: %s", s.mode),
            Interrupt: false,
        }, nil
    }
}
// UpdateMode changes the permission mode
func (s *Service) UpdateMode(mode options.PermissionMode) {
    s.mode = mode
}
```
## Phase 3: Adapters (Infrastructure)
### 3.1 CLI Transport Adapter (adapters/cli/transport.go)
Priority: Critical
This adapter implements the Transport port using subprocess CLI.
```go
package cli
import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "github.com/conneroisu/claude/pkg/claude/ports"
    "github.com/conneroisu/claude/pkg/claude/options"
)
// Adapter implements ports.Transport using CLI subprocess
type Adapter struct {
    options            *options.AgentOptions
    cliPath            string
    cmd                *exec.Cmd
    stdin              io.WriteCloser
    stdout             io.ReadCloser
    stderr             io.ReadCloser
    ready              bool
    exitErr            error
    closeStdinAfterWrite bool  // For one-shot queries
    mu                 sync.RWMutex
    maxBufferSize      int
}
// Verify interface compliance at compile time
var _ ports.Transport = (*Adapter)(nil)
const defaultMaxBufferSize = 1024 * 1024 // 1MB
func NewAdapter(opts *options.AgentOptions) *Adapter {
    maxBuf := defaultMaxBufferSize
    if opts.MaxBufferSize != nil {
        maxBuf = *opts.MaxBufferSize
    }
    return &Adapter{
        options:       opts,
        maxBufferSize: maxBuf,
    }
}
// findCLI locates the Claude CLI binary
func (a *Adapter) findCLI() (string, error) {
    // Check PATH first
    if path, err := exec.LookPath("claude"); err == nil {
        return path, nil
    }
    // Check common installation locations
    homeDir, _ := os.UserHomeDir()
    locations := []string{
        filepath.Join(homeDir, ".npm-global", "bin", "claude"),
        "/usr/local/bin/claude",
        filepath.Join(homeDir, ".local", "bin", "claude"),
        filepath.Join(homeDir, "node_modules", ".bin", "claude"),
        filepath.Join(homeDir, ".yarn", "bin", "claude"),
    }
    for _, loc := range locations {
        if _, err := os.Stat(loc); err == nil {
            return loc, nil
        }
    }
    return "", fmt.Errorf("claude CLI not found in PATH or common locations")
}
// BuildCommand constructs the CLI command with all options
// Exported for testing purposes
func (a *Adapter) BuildCommand() ([]string, error) {
    cmd := []string{a.cliPath, "--output-format", "stream-json", "--verbose"}
    // System prompt
    if a.options.SystemPrompt != nil {
        switch sp := a.options.SystemPrompt.(type) {
        case options.StringSystemPrompt:
            cmd = append(cmd, "--system-prompt", string(sp))
        case options.PresetSystemPrompt:
            if sp.Append != nil {
                cmd = append(cmd, "--append-system-prompt", *sp.Append)
            }
        }
    }
    // Tools
    if len(a.options.AllowedTools) > 0 {
        cmd = append(cmd, "--allowedTools", strings.Join(a.options.AllowedTools, ","))
    }
    if len(a.options.DisallowedTools) > 0 {
        cmd = append(cmd, "--disallowedTools", strings.Join(a.options.DisallowedTools, ","))
    }
    // Model and turns
    if a.options.Model != nil {
        cmd = append(cmd, "--model", *a.options.Model)
    }
    if a.options.MaxTurns != nil {
        cmd = append(cmd, "--max-turns", fmt.Sprintf("%d", *a.options.MaxTurns))
    }
    // Permissions
    if a.options.PermissionMode != nil {
        cmd = append(cmd, "--permission-mode", string(*a.options.PermissionMode))
    }
    if a.options.PermissionPromptToolName != nil {
        cmd = append(cmd, "--permission-prompt-tool", *a.options.PermissionPromptToolName)
    }
    // Session
    if a.options.ContinueConversation {
        cmd = append(cmd, "--continue")
    }
    if a.options.Resume != nil {
        cmd = append(cmd, "--resume", *a.options.Resume)
    }
    if a.options.ForkSession {
        cmd = append(cmd, "--fork-session")
    }
    // Settings
    if a.options.Settings != nil {
        cmd = append(cmd, "--settings", *a.options.Settings)
    }
    if len(a.options.SettingSources) > 0 {
        sources := make([]string, len(a.options.SettingSources))
        for i, s := range a.options.SettingSources {
            sources[i] = string(s)
        }
        cmd = append(cmd, "--setting-sources", strings.Join(sources, ","))
    }
    // Directories
    for _, dir := range a.options.AddDirs {
        cmd = append(cmd, "--add-dir", dir)
    }
    // MCP servers (configuration only, instances handled separately)
    if len(a.options.MCPServers) > 0 {
        // Convert to JSON config
        mcpConfig := map[string]any{"mcpServers": a.options.MCPServers}
        jsonBytes, err := json.Marshal(mcpConfig)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
        }
        cmd = append(cmd, "--mcp-config", string(jsonBytes))
    }
    // Extra arguments
    for flag, value := range a.options.ExtraArgs {
        if value == nil {
            cmd = append(cmd, "--"+flag)
        } else {
            cmd = append(cmd, "--"+flag, *value)
        }
    }
    return cmd, nil
}
func (a *Adapter) Connect(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.ready {
        return nil
    }
    // Find CLI
    cliPath, err := a.findCLI()
    if err != nil {
        return fmt.Errorf("CLI discovery failed: %w", err)
    }
    a.cliPath = cliPath
    // Build command
    cmdArgs, err := a.BuildCommand()
    if err != nil {
        return fmt.Errorf("command construction failed: %w", err)
    }
    // Set up environment
    env := os.Environ()
    env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")
    for k, v := range a.options.Env {
        env = append(env, k+"="+v)
    }
    // Create command
    a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
    a.cmd.Env = env
    if a.options.Cwd != nil {
        a.cmd.Dir = *a.options.Cwd
    }
    // Set up pipes
    stdin, err := a.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("stdin pipe failed: %w", err)
    }
    a.stdin = stdin
    stdout, err := a.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("stdout pipe failed: %w", err)
    }
    a.stdout = stdout
    stderr, err := a.cmd.StderrPipe()
    if err != nil {
        return fmt.Errorf("stderr pipe failed: %w", err)
    }
    a.stderr = stderr
    // Start process
    if err := a.cmd.Start(); err != nil {
        return fmt.Errorf("process start failed: %w", err)
    }
    // Start stderr handler if callback is set
    if a.options.StderrCallback != nil {
        go a.handleStderr()
    }
    // Detect one-shot mode: _isStreaming flag set by domain services
    // In one-shot mode, stdin should be closed after first write
    if !a.options._isStreaming {
        a.closeStdinAfterWrite = true
    }
    a.ready = true
    return nil
}
func (a *Adapter) handleStderr() {
    scanner := bufio.NewScanner(a.stderr)
    for scanner.Scan() {
        line := scanner.Text()
        if a.options.StderrCallback != nil {
            a.options.StderrCallback(line)
        }
    }
}
func (a *Adapter) Write(ctx context.Context, data string) error {
    a.mu.RLock()
    shouldClose := a.closeStdinAfterWrite
    a.mu.RUnlock()
    a.mu.Lock()
    defer a.mu.Unlock()
    if !a.ready {
        return fmt.Errorf("transport not ready")
    }
    if a.exitErr != nil {
        return fmt.Errorf("transport has exited: %w", a.exitErr)
    }
    _, err := a.stdin.Write([]byte(data))
    if err != nil {
        return err
    }
    // Close stdin after write for one-shot queries
    if shouldClose {
        a.closeStdinAfterWrite = false
        a.stdin.Close()
    }
    return nil
}
func (a *Adapter) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
    msgCh := make(chan map[string]any, 10)
    errCh := make(chan error, 1)
    go func() {
        defer close(msgCh)
        defer close(errCh)
        scanner := bufio.NewScanner(a.stdout)
        // Configure scanner buffer to handle large Claude responses
        // Default is 64KB which is insufficient for large responses
        scanBuf := make([]byte, 64*1024)
        scanner.Buffer(scanBuf, a.maxBufferSize)
        buffer := ""
        for scanner.Scan() {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            default:
            }
            line := scanner.Text()
            buffer += line
            // Check buffer size
            if len(buffer) > a.maxBufferSize {
                errCh <- fmt.Errorf("message buffer exceeded %d bytes", a.maxBufferSize)
                return
            }
            // Try to parse JSON
            var msg map[string]any
            if err := json.Unmarshal([]byte(buffer), &msg); err == nil {
                buffer = ""
                msgCh <- msg
            }
            // Continue buffering if incomplete
        }
        if err := scanner.Err(); err != nil {
            errCh <- err
        }
        // Check exit status
        if a.cmd != nil {
            if err := a.cmd.Wait(); err != nil {
                errCh <- fmt.Errorf("process exited with error: %w", err)
            }
        }
    }()
    return msgCh, errCh
}
func (a *Adapter) EndInput() error {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.stdin != nil {
        return a.stdin.Close()
    }
    return nil
}
func (a *Adapter) Close() error {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.ready = false
    // Close stdin
    if a.stdin != nil {
        a.stdin.Close()
    }
    // Terminate process
    if a.cmd != nil && a.cmd.Process != nil {
        a.cmd.Process.Kill()
        a.cmd.Wait()
    }
    return nil
}
func (a *Adapter) IsReady() bool {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.ready
}
```
### 3.2 JSON-RPC Protocol Adapter (adapters/jsonrpc/protocol.go)
Priority: High
Key Design: This adapter implements `ports.ProtocolHandler` and manages all control protocol state (pending requests, request IDs, etc.). The domain services delegate this infrastructure concern to the adapter.
```go
package jsonrpc
import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "sync"
    "time"
    "github.com/conneroisu/claude/pkg/claude/ports"
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/permissions"
)
// Adapter implements ports.ProtocolHandler for control protocol
// This is an INFRASTRUCTURE adapter - it handles protocol state management
type Adapter struct {
    transport      ports.Transport
    // Control protocol state (managed by adapter, not domain)
    pendingReqs    map[string]chan result
    requestCounter int
    mu             sync.Mutex
}
// Verify interface compliance at compile time
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
// Initialize is a no-op - initialization happens implicitly in StartMessageRouter
func (a *Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
    return nil, nil
}
// SendControlRequest sends a control request and waits for response
// This method handles all request ID generation and timeout logic
func (a *Adapter) SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
    // Generate unique request ID: req_{counter}_{randomHex}
    a.mu.Lock()
    a.requestCounter++
    requestID := fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(4))
    a.mu.Unlock()
    // Create result channel for this request
    resCh := make(chan result, 1)
    a.mu.Lock()
    a.pendingReqs[requestID] = resCh
    a.mu.Unlock()
    // Build control request envelope
    controlReq := map[string]any{
        "type":       "control_request",
        "request_id": requestID,
        "request":    req,
    }
    // Send via transport
    reqBytes, err := json.Marshal(controlReq)
    if err != nil {
        a.mu.Lock()
        delete(a.pendingReqs, requestID)
        a.mu.Unlock()
        return nil, fmt.Errorf("marshal control request: %w", err)
    }
    if err := a.transport.Write(ctx, string(reqBytes)+"\n"); err != nil {
        a.mu.Lock()
        delete(a.pendingReqs, requestID)
        a.mu.Unlock()
        return nil, fmt.Errorf("write control request: %w", err)
    }
    // Wait for response with 60s timeout
    timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()
    select {
    case <-timeoutCtx.Done():
        a.mu.Lock()
        delete(a.pendingReqs, requestID)
        a.mu.Unlock()
        if timeoutCtx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("control request timeout: %s", req["subtype"])
        }
        return nil, timeoutCtx.Err()
    case res := <-resCh:
        if res.err != nil {
            return nil, res.err
        }
        return res.data, nil
    }
}
// HandleControlRequest routes inbound control requests by subtype
func (a *Adapter) HandleControlRequest(
    ctx context.Context,
    req map[string]any,
    perms *permissions.Service,
    hooks map[string]hooking.HookCallback,
    mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
    request, _ := req["request"].(map[string]any)
    subtype, _ := request["subtype"].(string)
    switch subtype {
    case "can_use_tool":
        return a.handleCanUseTool(ctx, request, perms)
    case "hook_callback":
        return a.handleHookCallback(ctx, request, hooks)
    case "mcp_message":
        return a.handleMCPMessage(ctx, request, mcpServers)
    default:
        return nil, fmt.Errorf("unsupported control request subtype: %s", subtype)
    }
}
// StartMessageRouter continuously reads transport and partitions messages
func (a *Adapter) StartMessageRouter(
    ctx context.Context,
    msgCh chan<- map[string]any,
    errCh chan<- error,
    perms *permissions.Service,
    hooks map[string]hooking.HookCallback,
    mcpServers map[string]ports.MCPServer,
) error {
    go func() {
        transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-transportMsgCh:
                if !ok {
                    return
                }
                msgType, _ := msg["type"].(string)
                switch msgType {
                case "control_response":
                    // Route to pending request
                    a.routeControlResponse(msg)
                case "control_request":
                    // Handle inbound control request
                    go a.handleControlRequestAsync(ctx, msg, perms, hooks, mcpServers)
                case "control_cancel_request":
                    // TODO: Implement cancellation support
                    continue
                default:
                    // Forward SDK messages to public stream
                    select {
                    case msgCh <- msg:
                    case <-ctx.Done():
                        return
                    }
                }
            case err := <-transportErrCh:
                if err != nil {
                    select {
                    case errCh <- err:
                    case <-ctx.Done():
                    }
                    return
                }
            }
        }
    }()
    return nil
}
// routeControlResponse routes control_response messages to pending requests
func (a *Adapter) routeControlResponse(msg map[string]any) {
    response, _ := msg["response"].(map[string]any)
    requestID, _ := response["request_id"].(string)
    a.mu.Lock()
    defer a.mu.Unlock()
    if ch, exists := a.pendingReqs[requestID]; exists {
        subtype, _ := response["subtype"].(string)
        if subtype == "error" {
            errorMsg, _ := response["error"].(string)
            ch <- result{err: fmt.Errorf("control error: %s", errorMsg)}
        } else {
            responseData, _ := response["response"].(map[string]any)
            ch <- result{data: responseData}
        }
        delete(a.pendingReqs, requestID)
    }
}
// handleControlRequestAsync handles inbound control requests asynchronously
// Dependencies (perms, hooks, mcpServers) must be passed by the domain service that starts the router
func (a *Adapter) handleControlRequestAsync(
    ctx context.Context,
    msg map[string]any,
    perms *permissions.Service,
    hooks map[string]hooking.HookCallback,
    mcpServers map[string]ports.MCPServer,
) {
    requestID, _ := msg["request_id"].(string)
    // Handle the request
    responseData, err := a.HandleControlRequest(ctx, msg, perms, hooks, mcpServers)
    // Build response
    var response map[string]any
    if err != nil {
        response = map[string]any{
            "type": "control_response",
            "response": map[string]any{
                "subtype":    "error",
                "request_id": requestID,
                "error":      err.Error(),
            },
        }
    } else {
        response = map[string]any{
            "type": "control_response",
            "response": map[string]any{
                "subtype":    "success",
                "request_id": requestID,
                "response":   responseData,
            },
        }
    }
    // Send response
    resBytes, _ := json.Marshal(response)
    a.transport.Write(ctx, string(resBytes)+"\n")
}
// handleCanUseTool handles can_use_tool control requests
func (a *Adapter) handleCanUseTool(ctx context.Context, request map[string]any, perms *permissions.Service) (map[string]any, error) {
    toolName, _ := request["tool_name"].(string)
    input, _ := request["input"].(map[string]any)
    // suggestions, _ := request["permission_suggestions"].([]any) // TODO: Use suggestions
    if perms == nil {
        return nil, fmt.Errorf("permissions callback not provided")
    }
    result, err := perms.CheckToolUse(ctx, toolName, input)
    if err != nil {
        return nil, err
    }
    // Convert PermissionResult to response format
    switch r := result.(type) {
    case *permissions.PermissionResultAllow:
        response := map[string]any{"allow": true}
        if r.UpdatedInput != nil {
            response["input"] = r.UpdatedInput
        }
        // TODO: Handle updatedPermissions when control protocol supports it
        return response, nil
    case *permissions.PermissionResultDeny:
        return map[string]any{
            "allow":  false,
            "reason": r.Message,
        }, nil
    default:
        return nil, fmt.Errorf("unknown permission result type")
    }
}
// handleHookCallback handles hook_callback control requests
func (a *Adapter) handleHookCallback(ctx context.Context, request map[string]any, hooks map[string]hooking.HookCallback) (map[string]any, error) {
    callbackID, _ := request["callback_id"].(string)
    input, _ := request["input"].(map[string]any)
    toolUseID, _ := request["tool_use_id"].(*string)
    callback, exists := hooks[callbackID]
    if !exists {
        return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
    }
    // Execute callback
    hookCtx := hooking.HookContext{} // TODO: Add signal support
    result, err := callback(input, toolUseID, hookCtx)
    if err != nil {
        return nil, err
    }
    return result, nil
}
// handleMCPMessage handles mcp_message control requests
func (a *Adapter) handleMCPMessage(ctx context.Context, request map[string]any, mcpServers map[string]ports.MCPServer) (map[string]any, error) {
    serverName, _ := request["server_name"].(string)
    mcpMessage, _ := request["message"].(map[string]any)
    server, exists := mcpServers[serverName]
    if !exists {
        return a.mcpErrorResponse(mcpMessage, -32601, fmt.Sprintf("Server '%s' not found", serverName)), nil
    }
    method, _ := mcpMessage["method"].(string)
    messageID := mcpMessage["id"]
    params := mcpMessage["params"]
    var result any
    var err error
    switch method {
    case "initialize":
        result, err = server.Initialize(ctx, params)
    case "tools/list":
        result, err = server.ListTools(ctx)
    case "tools/call":
        callParams, _ := params.(map[string]any)
        toolName, _ := callParams["name"].(string)
        args, _ := callParams["arguments"].(map[string]any)
        result, err = server.CallTool(ctx, toolName, args)
    case "notifications/initialized":
        err = server.HandleNotification(ctx, method, params)
        result = nil // Notifications have no result
    default:
        return a.mcpErrorResponse(mcpMessage, -32601, "Method not found"), nil
    }
    if err != nil {
        return a.mcpErrorResponse(mcpMessage, -32603, err.Error()), nil
    }
    return map[string]any{
        "mcp_response": map[string]any{
            "jsonrpc": "2.0",
            "id":      messageID,
            "result":  result,
        },
    }, nil
}
// mcpErrorResponse creates an MCP JSON-RPC error response
func (a *Adapter) mcpErrorResponse(message map[string]any, code int, msg string) map[string]any {
    return map[string]any{
        "mcp_response": map[string]any{
            "jsonrpc": "2.0",
            "id":      message["id"],
            "error": map[string]any{
                "code":    code,
                "message": msg,
            },
        },
    }
}
// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
    b := make([]byte, n)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```
### 3.3 Message Parser Adapter (adapters/parse/parser.go)
Priority: High
This adapter implements `ports.MessageParser`, converting raw JSON messages from the transport into typed domain messages.
```go
package parse
import (
    "fmt"
    "github.com/conneroisu/claude/pkg/claude/ports"
    "github.com/conneroisu/claude/pkg/claude/messages"
)
// Adapter implements ports.MessageParser
// This is an INFRASTRUCTURE adapter - handles low-level message parsing
type Adapter struct{}
// Verify interface compliance at compile time
var _ ports.MessageParser = (*Adapter)(nil)
func NewAdapter() *Adapter {
    return &Adapter{}
}
// Parse implements ports.MessageParser
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
    msgType, ok := data["type"].(string)
    if !ok {
        return nil, fmt.Errorf("message missing type field")
    }
    switch msgType {
    case "user":
        return a.parseUserMessage(data)
    case "assistant":
        return a.parseAssistantMessage(data)
    case "system":
        return a.parseSystemMessage(data)
    case "result":
        return a.parseResultMessage(data)
    case "stream_event":
        return a.parseStreamEvent(data)
    default:
        return nil, fmt.Errorf("unknown message type: %s", msgType)
    }
}
func (a *Adapter) parseUserMessage(data map[string]any) (messages.Message, error) {
    // TODO: Parse user message fields
    return &messages.UserMessage{}, nil
}
func (a *Adapter) parseSystemMessage(data map[string]any) (messages.Message, error) {
    // TODO: Parse system message fields
    return &messages.SystemMessage{}, nil
}
func (a *Adapter) parseResultMessage(data map[string]any) (messages.Message, error) {
    // TODO: Parse result message fields
    return &messages.ResultMessage{}, nil
}
func (a *Adapter) parseStreamEvent(data map[string]any) (messages.Message, error) {
    // TODO: Parse stream event fields
    return &messages.StreamEvent{}, nil
}
func (a *Adapter) parseAssistantMessage(data map[string]any) (messages.Message, error) {
    // Parse content blocks
    msg, _ := data["message"].(map[string]any)
    contentArray, _ := msg["content"].([]any)
    var blocks []messages.ContentBlock
    for _, item := range contentArray {
        block, _ := item.(map[string]any)
        blockType, _ := block["type"].(string)
        switch blockType {
        case "text":
            text, _ := block["text"].(string)
            blocks = append(blocks, messages.TextBlock{Text: text})
        case "thinking":
            thinking, _ := block["thinking"].(string)
            signature, _ := block["signature"].(string)
            blocks = append(blocks, messages.ThinkingBlock{
                Thinking:  thinking,
                Signature: signature,
            })
        case "tool_use":
            id, _ := block["id"].(string)
            name, _ := block["name"].(string)
            input, _ := block["input"].(map[string]any)
            blocks = append(blocks, messages.ToolUseBlock{
                ID:    id,
                Name:  name,
                Input: input,
            })
        case "tool_result":
            toolUseID, _ := block["tool_use_id"].(string)
            content := block["content"]
            isError, _ := block["is_error"].(*bool)
            blocks = append(blocks, messages.ToolResultBlock{
                ToolUseID: toolUseID,
                Content:   content,
                IsError:   isError,
            })
        }
    }
    model, _ := msg["model"].(string)
    parentToolUseID := getStringPtr(data, "parent_tool_use_id")
    return &messages.AssistantMessage{
        Content:         blocks,
        Model:           model,
        ParentToolUseID: parentToolUseID,
    }, nil
}
// Helper function for extracting optional string pointers
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
Priority: Critical
```go
package claude
import (
    "context"
    "github.com/conneroisu/claude/pkg/claude/querying"
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/adapters/cli"
    "github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
    "github.com/conneroisu/claude/pkg/claude/messages"
    "github.com/conneroisu/claude/pkg/claude/options"
)
// Query performs a one-shot query to Claude
// This is the main entry point that wires up domain services with adapters
func Query(ctx context.Context, prompt string, opts *options.AgentOptions, hooks map[HookEvent][]HookMatcher) (<-chan messages.Message, <-chan error) {
    if opts == nil {
        opts = &options.AgentOptions{}
    }
    // Wire up adapters (infrastructure layer)
    transport := cli.NewAdapter(opts)
    protocol := jsonrpc.NewAdapter(transport)
    // Create domain services
    var hookingService *hooking.Service
    if hooks != nil {
        hookingService = hooking.NewService(hooks)
    }
    queryService := querying.NewService(transport, protocol, hookingService)
    // Execute domain logic
    return queryService.Execute(ctx, prompt, opts)
}
```
### 4.2 Client (client.go)
Priority: Critical
```go
package claude
import (
    "context"
    "github.com/conneroisu/claude/pkg/claude/streaming"
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/permissions"
    "github.com/conneroisu/claude/pkg/claude/adapters/cli"
    "github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
    "github.com/conneroisu/claude/pkg/claude/messages"
    "github.com/conneroisu/claude/pkg/claude/options"
    "sync"
)
// Client provides bidirectional, interactive conversations with Claude
// It's a facade that wires domain services with adapters
type Client struct {
    opts             *options.AgentOptions
    hooks            map[HookEvent][]HookMatcher
    permissions      *PermissionsConfig
    streamingService *streaming.Service
    mu               sync.Mutex
}
// NewClient creates a new Claude client
func NewClient(opts *options.AgentOptions, hooks map[HookEvent][]HookMatcher, perms *PermissionsConfig) *Client {
    if opts == nil {
        opts = &options.AgentOptions{}
    }
    return &Client{
        opts:        opts,
        hooks:       hooks,
        permissions: perms,
    }
}
// Connect establishes connection to Claude
func (c *Client) Connect(ctx context.Context, prompt any) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    // Wire up adapters (infrastructure)
    transport := cli.NewAdapter(c.opts)
    protocol := jsonrpc.NewAdapter(transport)
    // Wire up domain services
    var hookingService *hooking.Service
    if c.hooks != nil {
        hookingService = hooking.NewService(c.hooks)
    }
    var permissionsService *permissions.Service
    if c.permissions != nil {
        permissionsService = permissions.NewService(c.permissions)
    }
    // Create streaming service with dependencies
    c.streamingService = streaming.NewService(transport, protocol, hookingService, permissionsService)
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
Priority: Medium
The facade re-exports domain hook types from the `hooking` package for public API convenience:
```go
package claude
import (
    "github.com/conneroisu/claude/pkg/claude/hooking"
    "github.com/conneroisu/claude/pkg/claude/permissions"
)
// Re-export domain hook types for public API
type HookEvent = hooking.HookEvent
type HookContext = hooking.HookContext
type HookCallback = hooking.HookCallback
type HookMatcher = hooking.HookMatcher
// Re-export hook event constants
const (
    HookEventPreToolUse       = hooking.HookEventPreToolUse
    HookEventPostToolUse      = hooking.HookEventPostToolUse
    HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
    HookEventStop             = hooking.HookEventStop
    HookEventSubagentStop     = hooking.HookEventSubagentStop
    HookEventPreCompact       = hooking.HookEventPreCompact
)
// Re-export permissions types for public API
type PermissionsConfig = permissions.PermissionsConfig
type PermissionResult = permissions.PermissionResult
type CanUseToolFunc = permissions.CanUseToolFunc
type HookJSONOutput struct {
    Decision           *string        `json:"decision,omitempty"`           // "block"
    SystemMessage      *string        `json:"systemMessage,omitempty"`
    HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}
// Example hook implementation
func BlockBashPatternHook(patterns []string) HookCallback {
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
Priority: Medium
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
// CreateSDKMCPServer creates both config and server instance
// Config goes in AgentOptions.MCPServers, instance is registered separately
func CreateSDKMCPServer(name, version string, tools []*SDKMCPTool) (SDKServerConfig, MCPServer) {
    toolMap := make(map[string]*SDKMCPTool)
    for _, tool := range tools {
        toolMap[tool.Name] = tool
    }
    server := &sdkMCPServer{
        name:    name,
        version: version,
        tools:   toolMap,
    }
    config := SDKServerConfig{
        Type: "sdk",
        Name: name,
        // NO Instance field - that's the whole point!
    }
    return config, server
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
Priority: Medium
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
    "testing"
    "github.com/conneroisu/claude/pkg/claude/querying"
    "github.com/conneroisu/claude/pkg/claude/options"
)
// Mock transport implementing ports.Transport
type mockTransport struct {
    connectErr error
    messages   []map[string]any
}
func (m *mockTransport) Connect(ctx context.Context) error { return m.connectErr }
func (m *mockTransport) Write(ctx context.Context, data string) error { return nil }
// ... implement other methods
func TestService_Execute(t *testing.T) {
    tests := []struct {
        name       string
        prompt     string
        wantErr    bool
        setupMock  func(*mockTransport)
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
    "testing"
    "github.com/conneroisu/claude/pkg/claude/internal/parse"
    "github.com/conneroisu/claude/pkg/claude/messages"
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
    "testing"
    "github.com/conneroisu/claude/pkg/claude/adapters/cli"
    "github.com/conneroisu/claude/pkg/claude/options"
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
    "os"
    "testing"
    "time"
    "github.com/conneroisu/claude/pkg/claude"
    "github.com/conneroisu/claude/pkg/claude/options"
    "github.com/conneroisu/claude/pkg/claude/messages"
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
        "type": "result",
        "subtype": "success",
        "duration_ms": 1234,
        "num_turns": 1,
        "session_id": "test-session",
    }
)
```
### 6.4 CI/CD Test Automation
GitHub Actions Workflow:
```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.21', '1.22', '1.23']
    steps:
    - uses: actions/checkout@v4
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}
    - name: Run unit tests
      run: |
        go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.txt
  integration-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'
    - name: Install Claude CLI
      run: npm install -g @anthropic-ai/claude-code
    - name: Run integration tests
      env:
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
      run: |
        go test -tags=integration -v ./tests/integration/...
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v4
      with:
        version: latest
        args: --timeout=5m
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
    "log"
    "github.com/conneroisu/claude/pkg/claude"
    "github.com/conneroisu/claude/pkg/claude/messages"
    "github.com/conneroisu/claude/pkg/claude/options"
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
## Phase 7: Publishing & CI/CD
### 7.1 Go Module Setup
Priority: Critical
### 7.2 CI/CD Pipeline
Priority: High
### 7.3 Release Process
Priority: Medium
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
1. Domain Independence: Core domain packages (`querying`, `streaming`, `hooking`, `permissions`) never import adapters
2. Ports Define Contracts: Interfaces in `ports/` package are defined by domain needs, not external systems
3. Adapters Implement Ports: All infrastructure code in `adapters/` implements port interfaces
4. Dependency Direction: Always flows inward (adapters → domain), never outward (domain → adapters)
5. Package Naming: Named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)
### Go Idioms
6. Channels vs Iterators: Use channels for async message streaming (idiomatic Go)
7. Context Integration: Full context.Context support throughout
8. Error Handling: Return errors explicitly, use error wrapping
9. Interface Compliance: Use `var _ ports.Transport = (*Adapter)(nil)` pattern to verify at compile time
10. Async Model: Goroutines + channels (Go's native async)
11. JSON Handling: Use encoding/json with struct tags
12. Testing Strategy: Table-driven tests, interface mocks, integration tests
### Architectural Benefits
- Testability: Domain logic testable without infrastructure dependencies
- Flexibility: Easy to swap adapters (e.g., different transport mechanisms)
- Clarity: Clear separation between business logic and technical details
- Maintainability: Changes to infrastructure don't affect domain
- Discoverability: Package names describe purpose at a glance
## Success Criteria
1. Functional parity with Python SDK
2. Clean, idiomatic Go API
3. Efficient resource usage
4. Comprehensive documentation
5. >80% test coverage
6. Automated CI/CD
7. Easy to use and well-documented
## Hexagonal Architecture Summary
### Dependency Flow
The SDK strictly follows the dependency rule of hexagonal architecture:
```
┌─────────────────────────────────────────────────┐
│  LAYER 4: Public API (client.go, query.go)     │
│  - Entry point for SDK users                    │
│  - Wires domain services with adapters          │
└──────────────────┬──────────────────────────────┘
                   │ depends on ↓
┌──────────────────▼──────────────────────────────┐
│  LAYER 3: ADAPTERS (adapters/*)                 │
│  - cli/      → implements ports.Transport       │
│  - jsonrpc/  → implements ports.ProtocolHandler │
│  - parse/    → implements ports.MessageParser   │
│  - mcp/      → implements ports.MCPServer       │
└──────────────────┬──────────────────────────────┘
                   │ depends on ↓
┌──────────────────▼──────────────────────────────┐
│  LAYER 2: PORTS (ports/*)                       │
│  - Interfaces defined BY domain needs           │
│  - Contract layer between domain and infra      │
└──────────────────┬──────────────────────────────┘
                   │ depends on ↓
┌──────────────────▼──────────────────────────────┐
│  LAYER 1: CORE DOMAIN (querying/, streaming/)  │
│  - Pure business logic                          │
│  - No infrastructure dependencies               │
│  - Uses port interfaces, never adapters         │
└─────────────────────────────────────────────────┘
```
### Key Architectural Decisions
#### 1. Ports Define Contracts
All interfaces are defined in `ports/` based on domain needs:
- `ports.Transport` - What domain needs for I/O
- `ports.ProtocolHandler` - What domain needs for control protocol
- `ports.MessageParser` - What domain needs for parsing
- `ports.MCPServer` - What domain needs from MCP servers
#### 2. Adapters Implement Ports
Infrastructure code implements these interfaces:
- `adapters/cli.Adapter` implements `ports.Transport`
- `adapters/jsonrpc.Adapter` implements `ports.ProtocolHandler`
- `adapters/parse.Adapter` implements `ports.MessageParser`
- `adapters/mcp.Adapter` implements `ports.MCPServer`
#### 3. Domain Services Are Pure
Domain services contain ONLY business logic:
- `querying.Service` - Executes one-shot queries
- `streaming.Service` - Manages bidirectional conversations
- `hooking.Service` - Executes lifecycle hooks
- `permissions.Service` - Checks tool permissions
NO protocol state management (request IDs, timeouts, etc.) in domain.
#### 4. Infrastructure Concerns Stay in Adapters
Control protocol state management is in `adapters/jsonrpc`:
- Pending request tracking
- Request ID generation
- Timeout handling
- Response routing
#### 5. Configuration Separation
Options are split by concern:
- `options/domain.go` - Pure domain config (PermissionMode, AgentDefinition)
- `options/transport.go` - Infrastructure config (Cwd, Env, MaxBufferSize)
- `options/mcp.go` - MCP server configurations
### Benefits of This Architecture
1. Testability
   - Domain services testable without infrastructure
   - Mock adapters via interfaces
   - No subprocess spawning in unit tests
2. Flexibility
   - Swap CLI transport for HTTP transport
   - Change JSON-RPC to gRPC
   - Add new message parsers
3. Clarity
   - Clear boundaries between layers
   - Easy to understand where code belongs
   - Package names describe purpose
4. Maintainability
   - Infrastructure changes don't affect domain
   - Domain changes don't ripple to all adapters
   - Each layer has single responsibility
### Compile-Time Guarantees
All adapters verify interface compliance at compile time:
```go
// adapters/cli/transport.go
var _ ports.Transport = (*Adapter)(nil)
// adapters/jsonrpc/protocol.go
var _ ports.ProtocolHandler = (*Adapter)(nil)
// adapters/parse/parser.go
var _ ports.MessageParser = (*Adapter)(nil)
```
If an adapter doesn't fully implement its port interface, the code won't compile.
## Conclusion
This plan provides a comprehensive roadmap for implementing a production-ready Claude Agent SDK for Go following strict hexagonal architecture principles. The implementation:
- Separates domain logic from infrastructure
- Uses interfaces (ports) to define contracts
- Implements infrastructure via adapters
- Enforces dependency direction (inward only)
- Follows Go idioms and best practices
- Maintains functional parity with Python SDK
Next Steps: Begin implementation with Phase 1 (Core Domain & Ports), then Phase 2 (Domain Services), Phase 3 (Adapters), and finally Phase 4 (Public API).
