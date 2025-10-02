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
		mcpServers:  mcpServers,
		msgCh:       make(chan map[string]any),
		errCh:       make(chan error, 1),
	}
}

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