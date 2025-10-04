## Phase 2: Domain Services

### Control Protocol Architecture

Before implementing domain services, it's critical to understand how bidirectional communication works between the SDK and Claude CLI. This understanding shapes how domain services are designed.

#### Overview

The SDK uses a JSON-RPC control protocol layered on top of the transport (stdin/stdout). There are **three types of messages**:

1. **SDK Messages** - Regular messages (user, assistant, system, result, stream_event)
2. **Control Requests** - Bidirectional control messages (`type: "control_request"`)
3. **Control Responses** - Responses to control requests (`type: "control_response"`)

#### Message Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│ Domain Service Layer                                                  │
│  - Querying/Streaming services focus on business logic               │
│  - Delegate control protocol details to protocol adapter             │
└──────────────────────────────────────────────────────────────────────┘
                              │ ▲
                              │ │
                    SDK Messages + Control Requests/Responses
                              │ │
                              ▼ │
┌──────────────────────────────────────────────────────────────────────┐
│ Protocol Handler (jsonrpc adapter)                                   │
│  - Routes messages by type                                           │
│  - Manages pending requests (map[requestID]chan)                     │
│  - Tracks callback IDs for hooks                                     │
│  - Handles 60s timeouts on outbound requests                         │
│  - Generates request IDs: req_{counter}_{randomHex(4)}               │
└──────────────────────────────────────────────────────────────────────┘
                              │ ▲
                              │ │
                       JSON lines over stdin/stdout
                              │ │
                              ▼ │
┌──────────────────────────────────────────────────────────────────────┐
│ Transport Adapter (CLI subprocess)                                    │
│  - Manages subprocess and pipes                                       │
│  - Buffers partial JSON until complete                               │
│  - Handles process lifecycle                                         │
└──────────────────────────────────────────────────────────────────────┘
                              │ ▲
                              │ │
                              ▼ │
                      ┌──────────────┐
                      │  Claude CLI  │
                      └──────────────┘
```

#### Control Request Types

**SDK → CLI (Outbound):**
- `interrupt` - Stop current operation
- `set_permission_mode` - Change permission mode
- `set_model` - Change AI model
- `initialize` - Setup hooks (streaming only)

**CLI → SDK (Inbound):**
- `can_use_tool` - Ask permission for tool use
- `hook_callback` - Execute registered hook
- `mcp_message` - Proxy message to/from MCP server

#### Request ID Generation

Every control request needs a unique ID for response routing:

```go
// Pattern: req_{counter}_{randomHex}
requestID := fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(4))
a.requestCounter++

// Examples:
// "req_1_a3f2"
// "req_2_b8c4"
// "req_3_d1e9"

func randomHex(n int) string {
    b := make([]byte, n)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

#### Hook Callback ID Registration

During streaming initialization, hooks are registered with generated callback IDs:

```go
// Initialize request structure:
{
  "type": "control_request",
  "request_id": "req_1_a3f2",
  "request": {
    "subtype": "initialize",
    "hooks": {
      "PreToolUse": [
        {
          "matcher": "Bash",
          "hookCallbackIds": ["hook_0", "hook_1"]  // Generated IDs
        }
      ],
      "PostToolUse": [
        {
          "matcher": "*",
          "hookCallbackIds": ["hook_2"]
        }
      ]
    }
  }
}
```

**Hook ID Generation Pattern:**

```go
// During initialization, generate IDs for all hook callbacks
callbackID := fmt.Sprintf("hook_%d", nextCallbackID)
nextCallbackID++
hookCallbacksMap[callbackID] = userCallback

// When CLI sends hook_callback request:
// {
//   "subtype": "hook_callback",
//   "callback_id": "hook_0",  // <-- References our generated ID
//   "input": {...},
//   "tool_use_id": "toolu_xxx"
// }
//
// SDK looks up hookCallbacksMap["hook_0"] and invokes it
```

#### Permission Flow with Suggestions

When CLI requests permission, it includes suggestions for "always allow" workflows:

```go
// Inbound permission request from CLI:
{
  "subtype": "can_use_tool",
  "tool_name": "Bash",
  "input": {"command": "git status"},
  "permission_suggestions": [
    {
      "type": "addRules",
      "rules": [{"toolName": "Bash", "ruleContent": "git:*"}],
      "behavior": "allow",
      "destination": "userSettings"
    }
  ],
  "blocked_path": "/home/user/project"
}

// SDK calls user's can_use_tool callback, passing suggestions
permCtx := ToolPermissionContext{
    Suggestions: request.PermissionSuggestions,
}
result, err := canUseTool(ctx, toolName, input, permCtx)

// If user accepts with "always allow":
return &PermissionResultAllow{
    UpdatedPermissions: permCtx.Suggestions,  // Return CLI's suggestions
}

// SDK sends response:
{
  "allow": true,
  "updated_permissions": [...]  // CLI's original suggestions
}
```

#### Timeouts

All outbound control requests have a **60-second timeout** to prevent deadlocks:

```go
timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
defer cancel()

select {
case <-timeoutCtx.Done():
    return nil, fmt.Errorf("control request timeout: %s", subtype)
case result := <-resultChan:
    return result, nil
}
```

---

### 2.1 Querying Service (querying/service.go)
Priority: Critical
The querying service encapsulates the domain logic for executing one-shot queries.

**Python SDK Parity Check:**
The Python SDK's `query()` function (`src/claude_agent/client.py`) follows this pattern:
1. Initialize client session
2. Send prompt
3. Stream messages until result message received
4. Close session automatically

The Go implementation mirrors this with channels instead of async generators:
1. Connect transport (equivalent to Python's session init)
2. Write prompt to transport
3. Stream messages via channels (equivalent to Python's `async for message in client`)
4. Domain service handles cleanup (equivalent to Python's context manager)

**Concurrency Model:**
The querying service uses Go's native concurrency primitives:
- **Goroutines:** Message routing runs in a background goroutine started by the protocol handler
- **Channels:** `msgCh` and `errCh` provide async message delivery (unbuffered for backpressure)
- **Context:** Enables cancellation propagation through the entire stack
- **Select:** Used in the main loop to multiplex between message, error, and cancellation channels

**Channel Wiring Details:**
```
User calls Execute()
    ↓
  Launches goroutine
    ↓
  Protocol.StartMessageRouter(ctx, routerMsgCh, routerErrCh, ...)
    ↓ (starts another goroutine)
  Transport.ReadMessages(ctx) → (transportMsgCh, transportErrCh)
    ↓ (message routing goroutine)
  Route messages by type:
    - control_response → pending request handlers
    - control_request → inbound request handlers
    - SDK messages → forward to routerMsgCh
    ↓
  Domain goroutine receives from routerMsgCh
    ↓
  Parser.Parse(rawMsg) → typed message
    ↓
  Send to user's msgCh
```

All goroutines respect context cancellation via `select` statements.

Key Design Decision: Control protocol state management (pending requests, callback IDs, request counters) is handled by the `jsonrpc` adapter, NOT by domain services. The domain only uses the port interface.

```go
package querying

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
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
		mcpServers:  mcpServers, // Both client and SDK MCP servers (already wrapped as adapters)
	}
}

// Note: The mcpServers map contains ports.MCPServer implementations.
// For SDK servers: These are ServerAdapter instances wrapping user's *mcp.Server
// For client servers: These are ClientAdapter instances with active MCP client sessions
// The protocol adapter uses this map to route control protocol mcp_message requests

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

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
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
		mcpServers:  mcpServers, // MCP servers passed to protocol for control request handling
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}

// Note: MCP servers are initialized by the public API layer before creating this service.
// The service receives already-connected adapters (both client and SDK types).
// When control protocol receives mcp_message requests, it uses this map for routing.

func (s *Service) Connect(ctx context.Context, prompt *string) error {
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
	"fmt"
)

// HookEvent represents different hook trigger points
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventNotification     HookEvent = "Notification"
	HookEventSessionStart     HookEvent = "SessionStart"
	HookEventSessionEnd       HookEvent = "SessionEnd"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"
)
// BaseHookInput contains fields common to all hook inputs
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// HookInput is a discriminated union of all hook input types
// The specific type can be determined by the HookEventName field
type HookInput interface {
	hookInput()
}

// PreToolUseHookInput is the input for PreToolUse hooks
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"` // Intentionally flexible - varies by tool
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"`    // Intentionally flexible - varies by tool
	ToolResponse  any    `json:"tool_response"` // Intentionally flexible - varies by tool
}

func (PostToolUseHookInput) hookInput() {}

// NotificationHookInput is the input for Notification hooks
type NotificationHookInput struct {
	BaseHookInput
	HookEventName string  `json:"hook_event_name"` // "Notification"
	Message       string  `json:"message"`
	Title         *string `json:"title,omitempty"`
}

func (NotificationHookInput) hookInput() {}

// UserPromptSubmitHookInput is the input for UserPromptSubmit hooks
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "UserPromptSubmit"
	Prompt        string `json:"prompt"`
}

func (UserPromptSubmitHookInput) hookInput() {}

// SessionStartSource represents the source of a session start
type SessionStartSource string

const (
	SessionStartSourceStartup SessionStartSource = "startup"
	SessionStartSourceResume  SessionStartSource = "resume"
	SessionStartSourceClear   SessionStartSource = "clear"
	SessionStartSourceCompact SessionStartSource = "compact"
)

// SessionStartHookInput is the input for SessionStart hooks
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionStart"
	Source        string `json:"source"`          // "startup" | "resume" | "clear" | "compact"
}

func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is the input for SessionEnd hooks
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionEnd"
	Reason        string `json:"reason"`          // Exit reason
}

func (SessionEndHookInput) hookInput() {}

// StopHookInput is the input for Stop hooks
type StopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "Stop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (StopHookInput) hookInput() {}

// SubagentStopHookInput is the input for SubagentStop hooks
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "SubagentStop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is the input for PreCompact hooks
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"` // "PreCompact"
	Trigger            string  `json:"trigger"`         // "manual" | "auto"
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

func (PreCompactHookInput) hookInput() {}

// HookContext provides context for hook execution
type HookContext struct {
	// Signal provides cancellation and timeout support via context
	// Hook implementations should check Signal.Done() for cancellation
	Signal context.Context
}

// HookCallback is a function that handles hook events
// Note: The input parameter is intentionally map[string]any at the callback level
// to allow the protocol adapter to pass raw JSON. Domain services should parse
// this into the appropriate HookInput type based on hook_event_name field.
type HookCallback func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error)

// HookMatcher defines when a hook should execute
type HookMatcher struct {
	Matcher string         // Pattern to match (e.g., tool name, event type)
	Hooks   []HookCallback // Callbacks to execute
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
	hookCtx := HookContext{
		Signal: ctx, // Pass context for cancellation support
	}

	for _, matcher := range matchers {
		// Check if matcher applies to this input
		if !s.matchesPattern(matcher.Matcher, input) {
			continue
		}

		for _, callback := range matcher.Hooks {
			// Check for cancellation before executing each hook
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

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

// matchesPattern checks if a hook matcher pattern applies to the given input
func (s *Service) matchesPattern(pattern string, input map[string]any) bool {
	// Empty matcher matches all events
	if pattern == "" {
		return true
	}

	// Wildcard matches all
	if pattern == "*" {
		return true
	}

	// For PreToolUse/PostToolUse hooks, match against tool_name
	if toolName, ok := input["tool_name"].(string); ok {
		// Exact match
		if pattern == toolName {
			return true
		}
	}

	// Pattern doesn't match
	return false
}
```
### 2.4 Permissions Service (permissions/service.go)
Priority: High
The permissions service handles tool permission checks and updates.
```go
package permissions

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// PermissionResult represents the outcome of a permission check
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow indicates tool use is allowed
type PermissionResultAllow struct {
	UpdatedInput       map[string]any     // Intentionally flexible - tool inputs vary by tool
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
// input is intentionally map[string]any as tool inputs vary by tool
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)

// PermissionsConfig holds permission service configuration
type PermissionsConfig struct {
	Mode       options.PermissionMode
	CanUseTool CanUseToolFunc
}

// Service manages tool permissions
type Service struct {
	mode       options.PermissionMode
	canUseTool CanUseToolFunc
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
// suggestions parameter comes from the control protocol's permission_suggestions field
func (s *Service) CheckToolUse(ctx context.Context, toolName string, input map[string]any, suggestions []PermissionUpdate) (PermissionResult, error) {
	// 1. Check permission mode
	switch s.mode {
	case options.PermissionModeBypassPermissions:
		// Always allow
		return &PermissionResultAllow{}, nil
	case options.PermissionModeDefault, options.PermissionModeAcceptEdits, options.PermissionModePlan:
		// 2. Call canUseTool callback if set
		if s.canUseTool != nil {
			// Pass suggestions from control protocol to callback
			// These suggestions can be used in "always allow" flow
			permCtx := ToolPermissionContext{
				Suggestions: suggestions,
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

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**All services require decomposition:**

**querying/ package:**
- ❌ Single `service.go` (300+ lines planned)
- ✅ Split into 5 files:
  - `service.go` - Service struct + constructor (60 lines)
  - `execute.go` - Execute implementation (80 lines)
  - `routing.go` - Message routing logic (70 lines)
  - `errors.go` - Error handling helpers (50 lines)
  - `state.go` - Execution state management (40 lines)

**streaming/ package:**
- ❌ Single `service.go` (350+ lines planned)
- ✅ Split into 6 files:
  - `service.go` - Service struct + constructor (50 lines)
  - `connect.go` - Connection logic (70 lines)
  - `send.go` - SendMessage implementation (60 lines)
  - `receive.go` - ReceiveMessages implementation (80 lines)
  - `lifecycle.go` - Lifecycle methods (50 lines)
  - `state.go` - State management (40 lines)

**hooking/ and permissions/ packages:**
- ✅ Can likely fit in 1-2 files each (under 175 lines)

### Complexity Hotspots (25 line limit, complexity limits)

**Function extraction required for:**
- Message routing switch statements → Extract handler map pattern
- Control protocol handling → Extract per-subtype handlers
- Hook execution logic → Extract hook executor helper
- Validation logic → Extract dedicated validators
- Error handling → Extract error wrapper functions

**Patterns to use:**
- Early returns to reduce nesting
- Handler maps instead of large switch statements
- Extracted validation functions
- Result structs to limit return values

### Checklist

- [ ] Cyclomatic complexity ≤ 15 per function
- [ ] Cognitive complexity ≤ 20 per function
- [ ] Max nesting depth ≤ 3 levels
- [ ] All functions ≤ 25 lines
- [ ] Use early return pattern
- [ ] Extract validation to separate functions
- [ ] Extract complex logic to helpers