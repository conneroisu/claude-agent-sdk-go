// Package hooking manages lifecycle hook execution.
//
// Hooks are user-defined callbacks that execute at various points in the
// agent lifecycle (e.g., before/after tool use, on errors, etc.).
package hooking

import "context"

// HookEvent represents different hook trigger points.
type HookEvent string

const (
	// HookEventPreToolUse fires before a tool is used.
	HookEventPreToolUse HookEvent = "PreToolUse"
	// HookEventPostToolUse fires after a tool is used.
	HookEventPostToolUse HookEvent = "PostToolUse"
	// HookEventUserPromptSubmit fires when user submits a prompt.
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	// HookEventNotification fires when a notification is shown.
	HookEventNotification HookEvent = "Notification"
	// HookEventSessionStart fires when a session starts.
	HookEventSessionStart HookEvent = "SessionStart"
	// HookEventSessionEnd fires when a session ends.
	HookEventSessionEnd HookEvent = "SessionEnd"
	// HookEventStop fires when the agent stops.
	HookEventStop HookEvent = "Stop"
	// HookEventSubagentStop fires when a subagent stops.
	HookEventSubagentStop HookEvent = "SubagentStop"
	// HookEventPreCompact fires before conversation compaction.
	HookEventPreCompact HookEvent = "PreCompact"
)

// BaseHookInput contains fields common to all hook inputs.
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// HookInput is a discriminated union of all hook input types.
//
// The specific type can be determined by the HookEventName field.
type HookInput interface {
	hookInput()
}

// PreToolUseHookInput is the input for PreToolUse hooks.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"` // Varies by tool
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"`    // Varies by tool
	ToolResponse  any    `json:"tool_response"` // Varies by tool
}

func (PostToolUseHookInput) hookInput() {}

// NotificationHookInput is the input for Notification hooks.
type NotificationHookInput struct {
	BaseHookInput
	HookEventName string  `json:"hook_event_name"` // "Notification"
	Message       string  `json:"message"`
	Title         *string `json:"title,omitempty"`
}

func (NotificationHookInput) hookInput() {}

// UserPromptSubmitHookInput is the input for UserPromptSubmit hooks.
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "UserPromptSubmit"
	Prompt        string `json:"prompt"`
}

func (UserPromptSubmitHookInput) hookInput() {}

// SessionStartSource represents the source of a session start.
type SessionStartSource string

const (
	// SessionStartSourceStartup indicates a new session.
	SessionStartSourceStartup SessionStartSource = "startup"
	// SessionStartSourceResume indicates resuming a session.
	SessionStartSourceResume SessionStartSource = "resume"
	// SessionStartSourceClear indicates clearing conversation.
	SessionStartSourceClear SessionStartSource = "clear"
	// SessionStartSourceCompact indicates post-compaction restart.
	SessionStartSourceCompact SessionStartSource = "compact"
)

// SessionStartHookInput is the input for SessionStart hooks.
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionStart"
	Source        string `json:"source"`
}

func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is the input for SessionEnd hooks.
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionEnd"
	Reason        string `json:"reason"`
}

func (SessionEndHookInput) hookInput() {}

// StopHookInput is the input for Stop hooks.
type StopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "Stop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (StopHookInput) hookInput() {}

// SubagentStopHookInput is the input for SubagentStop hooks.
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "SubagentStop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is the input for PreCompact hooks.
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"` // "PreCompact"
	Trigger            string  `json:"trigger"`
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

func (PreCompactHookInput) hookInput() {}

// HookContext provides context for hook execution.
type HookContext struct {
	Signal context.Context
}

// HookCallback is a function that handles hook events.
//
// Input is intentionally map[string]any at the callback level to allow
// the protocol adapter to pass raw JSON. Domain services should parse
// this into the appropriate HookInput type based on hook_event_name field.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

// HookMatcher defines when a hook should execute.
type HookMatcher struct {
	Matcher string         // Pattern to match (e.g., tool name)
	Hooks   []HookCallback // Callbacks to execute
}
