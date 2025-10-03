package hooking

// PreToolUseHookInput is the input for PreToolUse hooks.
type PreToolUseHookInput struct {
	BaseHookInput

	// HookEventName is always "PreToolUse"
	HookEventName string `json:"hook_event_name"`

	// ToolName identifies the tool being invoked
	ToolName string `json:"tool_name"`

	// ToolInput contains tool parameters.
	// Intentionally flexible - varies by tool.
	ToolInput any `json:"tool_input"`
}

// hookInput implements the HookInput interface.
func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is the input for PostToolUse hooks.
type PostToolUseHookInput struct {
	BaseHookInput

	// HookEventName is always "PostToolUse"
	HookEventName string `json:"hook_event_name"`

	// ToolName identifies the tool that was invoked
	ToolName string `json:"tool_name"`

	// ToolInput contains the tool parameters that were used.
	// Intentionally flexible - varies by tool.
	ToolInput any `json:"tool_input"`

	// ToolResponse contains the tool's output.
	// Intentionally flexible - varies by tool.
	ToolResponse any `json:"tool_response"`
}

// hookInput implements the HookInput interface.
func (PostToolUseHookInput) hookInput() {}

// NotificationHookInput is the input for Notification hooks.
type NotificationHookInput struct {
	BaseHookInput

	// HookEventName is always "Notification"
	HookEventName string `json:"hook_event_name"`

	// Message is the notification content
	Message string `json:"message"`

	// Title is optional notification title
	Title *string `json:"title,omitempty"`
}

// hookInput implements the HookInput interface.
func (NotificationHookInput) hookInput() {}

// UserPromptSubmitHookInput is the input for UserPromptSubmit
// hooks.
type UserPromptSubmitHookInput struct {
	BaseHookInput

	// HookEventName is always "UserPromptSubmit"
	HookEventName string `json:"hook_event_name"`

	// Prompt is the user's input text
	Prompt string `json:"prompt"`
}

// hookInput implements the HookInput interface.
func (UserPromptSubmitHookInput) hookInput() {}

// SessionStartSource represents the source of a session start.
type SessionStartSource string

const (
	// SessionStartSourceStartup indicates initial startup.
	SessionStartSourceStartup SessionStartSource = "startup"

	// SessionStartSourceResume indicates session resume.
	SessionStartSourceResume SessionStartSource = "resume"

	// SessionStartSourceClear indicates session clear.
	SessionStartSourceClear SessionStartSource = "clear"

	// SessionStartSourceCompact indicates post-compaction.
	SessionStartSourceCompact SessionStartSource = "compact"
)

// SessionStartHookInput is the input for SessionStart hooks.
type SessionStartHookInput struct {
	BaseHookInput

	// HookEventName is always "SessionStart"
	HookEventName string `json:"hook_event_name"`

	// Source indicates why the session started
	// ("startup", "resume", "clear", "compact")
	Source string `json:"source"`
}

// hookInput implements the HookInput interface.
func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is the input for SessionEnd hooks.
type SessionEndHookInput struct {
	BaseHookInput

	// HookEventName is always "SessionEnd"
	HookEventName string `json:"hook_event_name"`

	// Reason indicates why the session ended
	Reason string `json:"reason"`
}

// hookInput implements the HookInput interface.
func (SessionEndHookInput) hookInput() {}

// StopHookInput is the input for Stop hooks.
type StopHookInput struct {
	BaseHookInput

	// HookEventName is always "Stop"
	HookEventName string `json:"hook_event_name"`

	// StopHookActive indicates if stop hook is active
	StopHookActive bool `json:"stop_hook_active"`
}

// hookInput implements the HookInput interface.
func (StopHookInput) hookInput() {}

// SubagentStopHookInput is the input for SubagentStop hooks.
type SubagentStopHookInput struct {
	BaseHookInput

	// HookEventName is always "SubagentStop"
	HookEventName string `json:"hook_event_name"`

	// StopHookActive indicates if stop hook is active
	StopHookActive bool `json:"stop_hook_active"`
}

// hookInput implements the HookInput interface.
func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is the input for PreCompact hooks.
type PreCompactHookInput struct {
	BaseHookInput

	// HookEventName is always "PreCompact"
	HookEventName string `json:"hook_event_name"`

	// Trigger indicates compaction type ("manual" or "auto")
	Trigger string `json:"trigger"`

	// CustomInstructions optionally provides compaction guidance
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// hookInput implements the HookInput interface.
func (PreCompactHookInput) hookInput() {}
