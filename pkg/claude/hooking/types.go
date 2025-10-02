// Package hooking provides a hook system for intercepting and responding to
// events in the Claude Agent SDK.
package hooking

import "context"

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
type HookInput interface {
	hookInput()
}

// PreToolUseHookInput is the input for PreToolUse hooks
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`
	// Intentionally flexible - varies by tool
	ToolInput any `json:"tool_input"`
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`
	// Intentionally flexible - varies by tool
	ToolInput any `json:"tool_input"`
	// Intentionally flexible - varies by tool
	ToolResponse any `json:"tool_response"`
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

// SessionStartHookInput is the input for SessionStart hooks
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionStart"
	// "startup" | "resume" | "clear" | "compact"
	Source string `json:"source"`
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
// Note: The input parameter is intentionally map[string]any at the callback
// level to allow the protocol adapter to pass raw JSON. Domain services
// should parse this into the appropriate HookInput type based on
// hook_event_name field.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

// HookMatcher defines when a hook should execute
type HookMatcher struct {
	Matcher string         // Pattern to match (e.g., tool name, event type)
	Hooks   []HookCallback // Callbacks to execute
}
