// Package hooking manages lifecycle hook execution for the
// Claude Agent SDK.
//
// Hooks allow users to intercept and modify agent behavior at
// specific lifecycle points (pre/post tool use, prompts, etc.).
package hooking

import "context"

// HookEvent represents different hook trigger points.
type HookEvent string

const (
	// HookEventPreToolUse fires before tool execution.
	HookEventPreToolUse HookEvent = "PreToolUse"

	// HookEventPostToolUse fires after tool execution.
	HookEventPostToolUse HookEvent = "PostToolUse"

	// HookEventUserPromptSubmit fires when user submits prompt.
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"

	// HookEventStop fires when agent stops.
	HookEventStop HookEvent = "Stop"

	// HookEventSubagentStop fires when subagent stops.
	HookEventSubagentStop HookEvent = "SubagentStop"

	// HookEventPreCompact fires before conversation compaction.
	HookEventPreCompact HookEvent = "PreCompact"
)

// HookInput is a discriminated union of all hook input types.
// The specific type can be determined by the HookEventName field.
type HookInput interface {
	hookInput()
}

// BaseHookInput contains fields common to all hook inputs.
type BaseHookInput struct {
	// SessionID identifies the conversation session
	SessionID string `json:"session_id"`

	// TranscriptPath is path to conversation transcript
	TranscriptPath string `json:"transcript_path"`

	// Cwd is the current working directory
	Cwd string `json:"cwd"`

	// PermissionMode indicates current permission settings
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// HookContext provides context for hook execution.
type HookContext struct {
	// Signal provides cancellation and timeout support via
	// context. Hook implementations should check Signal.Done()
	// for cancellation.
	Signal context.Context
}

// HookCallback is a function that handles hook events.
// Note: The input parameter is intentionally map[string]any at
// the callback level to allow the protocol adapter to pass raw
// JSON. Domain services should parse this into the appropriate
// HookInput type based on hook_event_name field.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

// HookMatcher defines when a hook should execute.
type HookMatcher struct {
	// Matcher is pattern to match (e.g., tool name, event type)
	Matcher string

	// Hooks are callbacks to execute when pattern matches
	Hooks []HookCallback
}
