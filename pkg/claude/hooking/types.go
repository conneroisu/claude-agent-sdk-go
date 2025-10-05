package hooking

// HookEvent represents a hook execution point in the agent lifecycle.
type HookEvent string

// Hook event constants.
const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"
	HookEventNotification     HookEvent = "Notification"
	HookEventSessionStart     HookEvent = "SessionStart"
	HookEventSessionEnd       HookEvent = "SessionEnd"
)

// HookCallback is a user-provided function that executes when a hook fires.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx HookContext,
) (map[string]any, error)

// HookMatcher pairs a pattern with a callback for event matching.
type HookMatcher struct {
	Pattern  string
	Callback HookCallback
}

// HookContext provides conversation context to hook callbacks.
type HookContext struct {
	SessionID      string
	TranscriptPath string
	Cwd            string
	PermissionMode string
}
