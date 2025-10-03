## Phase 1: Core Domain & Ports

### 1.1 Domain Models (messages/, options/)

Priority: Critical
Define core domain models that are free from infrastructure concerns:
**Design Decision: When to Use `map[string]any` vs Typed Structs**
This SDK strikes a balance between type safety and flexibility:
**Use Typed Structs When:**

- The structure is well-defined and stable (e.g., UsageStats, HookInput types)
- The SDK needs to access specific fields (e.g., ResultMessage fields)
- Type safety provides clear benefits (e.g., discriminated unions)
  **Use `map[string]any` When:**
- Data varies by context and cannot be predetermined (e.g., tool inputs)
- The SDK only passes data through without inspecting it (e.g., raw stream events)
- Flexibility is more important than compile-time validation (e.g., SystemMessage.Data at message level)
  **Examples in this SDK:**
- ✅ Typed: `HookInput` (9 different types), `ResultMessage` (2 variants), `UsageStats`
- ❌ Flexible: `ToolUseBlock.Input` (varies by tool), `StreamEvent.Event` (raw API events)
- 🔄 Hybrid: `SystemMessage.Data` is `map[string]any` but can be parsed into typed `SystemMessageData` variants
  messages/messages.go - Message Types:

```go
package messages

// Message types
type Message interface {
	message()
}

type UserMessage struct {
	Content         MessageContent `json:"content"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (UserMessage) message() {}

type AssistantMessage struct {
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) message() {}

type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"` // Flexible - parse into SystemMessageData based on Subtype
}

func (SystemMessage) message() {}

// SystemMessageData is a discriminated union for SystemMessage.Data
// Parse this from map[string]any based on Subtype field
type SystemMessageData interface {
	systemMessageData()
}

// SystemMessageInit is sent at the start of a session
type SystemMessageInit struct {
	Agents         []string          `json:"agents,omitempty"`
	APIKeySource   string            `json:"apiKeySource"`
	Cwd            string            `json:"cwd"`
	Tools          []string          `json:"tools"`
	MCPServers     []MCPServerStatus `json:"mcp_servers"`
	Model          string            `json:"model"`
	PermissionMode string            `json:"permissionMode"`
	SlashCommands  []string          `json:"slash_commands"`
	OutputStyle    string            `json:"output_style"`
}

func (SystemMessageInit) systemMessageData() {}

// SystemMessageCompactBoundary marks a conversation compaction point
type SystemMessageCompactBoundary struct {
	CompactMetadata struct {
		Trigger   string `json:"trigger"` // "manual" | "auto"
		PreTokens int    `json:"pre_tokens"`
	} `json:"compact_metadata"`
}

func (SystemMessageCompactBoundary) systemMessageData() {}

// MCPServerStatus represents the status of an MCP server
type MCPServerStatus struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "connected" | "failed" | "needs-auth" | "pending"
	ServerInfo *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
}

// ResultMessage is a discriminated union based on Subtype
// Common fields are embedded in both variants
type ResultMessage interface {
	resultMessage()
}

// ResultMessageSuccess indicates a successful query completion
type ResultMessageSuccess struct {
	Subtype           string                `json:"subtype"` // "success"
	DurationMs        int                   `json:"duration_ms"`
	DurationAPIMs     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	SessionID         string                `json:"session_id"`
	Result            string                `json:"result"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             UsageStats            `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"` // Model name -> usage stats
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
}

func (ResultMessageSuccess) resultMessage() {}
func (ResultMessageSuccess) message()       {}

type MessageErrorSubtype string

const (
	MessageErrorSubtypeErrorMaxTurns        MessageErrorSubtype = "error_max_turns"
	MessageErrorSubtypeErrorDuringExecution MessageErrorSubtype = "error_during_execution"
)

// ResultMessageError indicates an error during execution
type ResultMessageError struct {
	Subtype           MessageErrorSubtype   `json:"subtype"` // "error_max_turns" | "error_during_execution"
	DurationMs        int                   `json:"duration_ms"`
	DurationAPIMs     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	SessionID         string                `json:"session_id"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             UsageStats            `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
}

func (ResultMessageError) resultMessage() {}
func (ResultMessageError) message()       {}

type StreamEvent struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Event           map[string]any `json:"event"` // Flexible - raw Anthropic API stream event
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) message() {}

// Content blocks
type ContentBlock interface {
	contentBlock()
}

type TextBlock struct {
	Type string `json:"type"` // Always "text"
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

type ThinkingBlock struct {
	Type      string `json:"type"` // Always "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

func (ThinkingBlock) contentBlock() {}

type ToolUseBlock struct {
	Type  string         `json:"type"` // Always "tool_use"
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"` // Flexible - tool inputs vary by tool
}

func (ToolUseBlock) contentBlock() {}

// ToolResultContent can be string or a list of content blocks as maps
type ToolResultContent interface {
	toolResultContent()
}

type ToolResultStringContent string
type ToolResultBlockListContent []map[string]any

func (ToolResultStringContent) toolResultContent()      {}
func (ToolResultBlockListContent) toolResultContent() {}

type ToolResultBlock struct {
	Type      string            `json:"type"` // Always "tool_result"
	ToolUseID string            `json:"tool_use_id"`
	Content   ToolResultContent `json:"content"` // Can be string or []ContentBlock
	IsError   *bool             `json:"is_error,omitempty"`
}

func (ToolResultBlock) contentBlock() {}

// Message content can be string or []ContentBlock
type MessageContent interface {
	messageContent()
}

type StringContent string
type BlockListContent []ContentBlock

func (StringContent) messageContent()    {}
func (BlockListContent) messageContent() {}

// UsageStats represents API usage statistics
type UsageStats struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ModelUsage represents usage statistics for a specific model
type ModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
}

// PermissionDenial represents a tool use that was denied by permissions
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"` // Intentionally flexible - varies by tool
}
```

options/domain.go - Pure Domain Configuration:

```go name="options/domain.go"
package options

// PermissionMode defines how permissions are handled
// This is a DOMAIN concept - it affects business logic
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModeAsk               PermissionMode = "ask"
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
	Name         string
	Description  string
	SystemPrompt *string
	AllowedTools []string
	Model        *string
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
	ContinueConversation   bool
	Resume                 *string
	ForkSession            bool
	IncludePartialMessages bool
	// Infrastructure settings (how to connect/execute)
	Cwd            *string
	Settings       *string
	AddDirs        []string
	Env            map[string]string
	User           *string
	SettingSources []SettingSource
	MaxBufferSize  *int
	StderrCallback func(string)
	ExtraArgs      map[string]*string
	// MCP server configuration (infrastructure)
	MCPServers map[string]MCPServerConfig
	// Internal flags (set by domain services, not by users)
	_isStreaming bool // Internal: true for Client, false for Query
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
	Initialize(ctx context.Context, config map[string]any) (map[string]any, error)
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

// MCPServer defines an interface for an in-process MCP Server.
// It abstracts the underlying implementation, which should be a wrapper around
// the official MCP Go SDK (github.com/modelcontextprotocol/go-sdk).
// This allows the agent to route raw MCP messages from the Claude CLI
// to a user-defined tool server.
type MCPServer interface {
	Name() string
	// HandleMessage takes a raw JSON-RPC message, processes it, and returns
	// a raw JSON-RPC response. This is used to proxy messages.
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)
	// Close closes the MCP server connection and releases resources
	Close() error
}
```

### 1.3 Error Types (errors.go)

Priority: Critical

```go
package claude

var (
	ErrNotConnected   = errors.New("claude: not connected")
	ErrCLINotFound    = errors.New("claude: CLI not found")
	ErrCLIConnection  = errors.New("claude: connection failed")
	ErrProcessFailed  = errors.New("claude: process failed")
	ErrJSONDecode     = errors.New("claude: JSON decode failed")
	ErrMessageParse   = errors.New("claude: message parse failed")
	ErrControlTimeout = errors.New("claude: control request timeout")
	ErrInvalidInput   = errors.New("claude: invalid input")
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

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**messages/ package requires decomposition:**
- ❌ Single `messages.go` (500+ lines planned)
- ✅ Split into 8 files:
  - `messages.go` - Core interfaces (50 lines)
  - `user.go` - UserMessage (40 lines)
  - `assistant.go` - AssistantMessage (60 lines)
  - `system.go` - SystemMessage types (80 lines)
  - `result.go` - ResultMessage types (90 lines)
  - `stream.go` - StreamEvent (30 lines)
  - `content.go` - ContentBlock types (70 lines)
  - `usage.go` - Usage statistics (40 lines)

**Other files are compliant:**
- ✅ `options/domain.go` (80 lines)
- ✅ `options/transport.go` (90 lines)
- ✅ `options/mcp.go` (70 lines)
- ✅ `ports/*.go` (25-60 lines each)

### Complexity Hotspots

- Message type parsing → Extract per-type parsers
- Content block switching → Use type-specific functions
- Validation logic → Extract to dedicated functions

### Checklist

- [ ] All files under 175 lines (excl. comments/blanks)
- [ ] All functions under 25 lines
- [ ] Max 4 parameters per function
- [ ] Max 3 return values per function
- [ ] 15% minimum comment density per file
- [ ] All exported items have godoc comments
