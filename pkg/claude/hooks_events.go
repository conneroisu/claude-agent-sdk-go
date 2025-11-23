package claude

// HookEvent represents different hook events.
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventNotification     HookEvent = "Notification"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventSessionStart     HookEvent = "SessionStart"
	HookEventSessionEnd       HookEvent = "SessionEnd"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"

	// HookEventPermissionRequest is triggered when a tool requires permission before execution.
	HookEventPermissionRequest HookEvent = "PermissionRequest"

	// HookEventSubagentStart is triggered when a subagent begins execution.
	HookEventSubagentStart HookEvent = "SubagentStart"
)

// HookEvents is a slice of all valid hook events.
var HookEvents = []HookEvent{
	HookEventPreToolUse,
	HookEventPostToolUse,
	HookEventNotification,
	HookEventUserPromptSubmit,
	HookEventSessionStart,
	HookEventSessionEnd,
	HookEventStop,
	HookEventSubagentStop,
	HookEventPreCompact,
	HookEventPermissionRequest,
	HookEventSubagentStart,
}
