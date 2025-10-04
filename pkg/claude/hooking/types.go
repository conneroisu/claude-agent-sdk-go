// Package hooking manages lifecycle hook execution for Claude Agent SDK.
// Hooks allow users to intercept and respond to events during agent execution.
package hooking

import "context"

// HookEvent represents different hook trigger points.
type HookEvent string

const (
	// HookEventPreToolUse fires before a tool is executed.
	HookEventPreToolUse HookEvent = "PreToolUse"

	// HookEventPostToolUse fires after a tool executes.
	HookEventPostToolUse HookEvent = "PostToolUse"

	// HookEventUserPromptSubmit fires when user submits a prompt.
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"

	// HookEventNotification fires for system notifications.
	HookEventNotification HookEvent = "Notification"

	// HookEventSessionStart fires when a session begins.
	HookEventSessionStart HookEvent = "SessionStart"

	// HookEventSessionEnd fires when a session ends.
	HookEventSessionEnd HookEvent = "SessionEnd"

	// HookEventStop fires when execution stops.
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
// The specific type is determined by the HookEventName field.
type HookInput interface {
	hookInput()
}

// PreToolUseHookInput is the input for PreToolUse hooks.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"` // Flexible - varies by tool
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"`    // Flexible - varies by tool
	ToolResponse  any    `json:"tool_response"` // Flexible - varies by tool
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

	// SessionStartSourceResume indicates resuming an existing session.
	SessionStartSourceResume SessionStartSource = "resume"

	// SessionStartSourceClear indicates session was cleared.
	SessionStartSourceClear SessionStartSource = "clear"

	// SessionStartSourceCompact indicates session was compacted.
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
	// Signal provides cancellation and timeout support.
	// Hook implementations should check Signal.Done() for cancellation.
	Signal context.Context
}

// HookCallback is a function that handles hook events.
// Note: The input parameter is intentionally map[string]any to allow
// the protocol adapter to pass raw JSON.
//
//revive:disable:line-length-limit Long function signature required for callback
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

//revive:enable:line-length-limit

// HookMatcher defines when a hook should execute.
type HookMatcher struct {
	// Matcher is a pattern to match (e.g., tool name, event type)
	Matcher string

	// Hooks are callbacks to execute when matcher applies
	Hooks []HookCallback
}
