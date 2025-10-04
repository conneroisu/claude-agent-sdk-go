package hooking

import "context"

// HookEvent represents different hook trigger points throughout
// the agent's lifecycle. Each event corresponds to a specific
// interaction pattern between the SDK and Claude CLI.
type HookEvent string

const (
	// HookEventPreToolUse triggers before a tool is executed.
	HookEventPreToolUse HookEvent = "PreToolUse"
	// HookEventPostToolUse triggers after a tool completes.
	HookEventPostToolUse HookEvent = "PostToolUse"
	// HookEventUserPromptSubmit triggers when user submits a prompt.
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	// HookEventNotification triggers for system notifications.
	HookEventNotification HookEvent = "Notification"
	// HookEventSessionStart triggers at session initialization.
	HookEventSessionStart HookEvent = "SessionStart"
	// HookEventSessionEnd triggers at session termination.
	HookEventSessionEnd HookEvent = "SessionEnd"
	// HookEventStop triggers when execution stops.
	HookEventStop HookEvent = "Stop"
	// HookEventSubagentStop triggers when a subagent stops.
	HookEventSubagentStop HookEvent = "SubagentStop"
	// HookEventPreCompact triggers before transcript compaction.
	HookEventPreCompact HookEvent = "PreCompact"
)

// BaseHookInput contains fields common to all hook inputs.
// It provides session context, file paths, and permission state.
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// HookInput is a discriminated union of all hook input types.
// The specific type can be determined by the HookEventName field.
type HookInput interface {
	hookInput()
}

// PreToolUseHookInput is the input for PreToolUse hooks.
// It provides tool name and input before tool execution.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PreToolUse"
	ToolName      string `json:"tool_name"`
	// ToolInput is intentionally flexible as it varies by tool
	ToolInput any `json:"tool_input"`
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks.
// It provides tool name, input, and response after execution.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostToolUse"
	ToolName      string `json:"tool_name"`
	// ToolInput is intentionally flexible as it varies by tool
	ToolInput any `json:"tool_input"`
	// ToolResponse is intentionally flexible as it varies by tool
	ToolResponse any `json:"tool_response"`
}

func (PostToolUseHookInput) hookInput() {}

// NotificationHookInput is the input for Notification hooks.
// It provides notification message and optional title.
type NotificationHookInput struct {
	BaseHookInput
	HookEventName string  `json:"hook_event_name"` // "Notification"
	Message       string  `json:"message"`
	Title         *string `json:"title,omitempty"`
}

func (NotificationHookInput) hookInput() {}

// UserPromptSubmitHookInput is the input for UserPromptSubmit hooks.
// It provides the user's submitted prompt text.
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "UserPromptSubmit"
	Prompt        string `json:"prompt"`
}

func (UserPromptSubmitHookInput) hookInput() {}

// SessionStartSource represents the source of a session start.
type SessionStartSource string

const (
	// SessionStartSourceStartup indicates a fresh session start.
	SessionStartSourceStartup SessionStartSource = "startup"
	// SessionStartSourceResume indicates resuming an existing session.
	SessionStartSourceResume SessionStartSource = "resume"
	// SessionStartSourceClear indicates starting after clearing.
	SessionStartSourceClear SessionStartSource = "clear"
	// SessionStartSourceCompact indicates starting after compacting.
	SessionStartSourceCompact SessionStartSource = "compact"
)

// SessionStartHookInput is the input for SessionStart hooks.
// It indicates how the session was initiated.
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionStart"
	// Source can be "startup", "resume", "clear", or "compact"
	Source string `json:"source"`
}

func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is the input for SessionEnd hooks.
// It provides the reason for session termination.
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionEnd"
	Reason        string `json:"reason"`          // Exit reason
}

func (SessionEndHookInput) hookInput() {}

// StopHookInput is the input for Stop hooks.
// It indicates whether the stop hook is currently active.
type StopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "Stop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (StopHookInput) hookInput() {}

// SubagentStopHookInput is the input for SubagentStop hooks.
// It indicates whether the stop hook is currently active.
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "SubagentStop"
	StopHookActive bool   `json:"stop_hook_active"`
}

func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is the input for PreCompact hooks.
// It provides compaction trigger and optional custom instructions.
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"` // "PreCompact"
	Trigger            string  `json:"trigger"`         // "manual" | "auto"
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

func (PreCompactHookInput) hookInput() {}

// HookContext provides context for hook execution.
// It enables cancellation and timeout support via standard context.
type HookContext struct {
	// Signal provides cancellation and timeout support via context.
	// Hook implementations should check Signal.Done() for cancellation.
	Signal context.Context
}

// HookCallback is a function that handles hook events.
// The input parameter is intentionally map[string]any to allow the
// protocol adapter to pass raw JSON. Domain services should parse
// this into the appropriate HookInput type based on hook_event_name.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

// HookMatcher defines when a hook should execute.
// It combines a pattern matcher with one or more callbacks.
type HookMatcher struct {
	Matcher string         // Pattern to match (e.g., tool name, event type)
	Hooks   []HookCallback // Callbacks to execute
}
