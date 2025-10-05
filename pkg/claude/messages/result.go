package messages

// ResultMessageSuccess represents a successful query completion.
// This message indicates that the conversation completed successfully
// and contains the final state and usage statistics.
type ResultMessageSuccess struct {
	// SessionID identifies the conversation session
	SessionID string

	// ConversationID identifies the conversation
	ConversationID string

	// Messages contains the full conversation history
	Messages []Message

	// Usage contains token usage statistics
	Usage *UsageStats

	// PermissionDenials contains any tools that were denied permission
	PermissionDenials []PermissionDenial

	// MCPServerStatuses contains the status of MCP servers
	MCPServerStatuses []MCPServerStatus
}

func (*ResultMessageSuccess) message()       {}
func (*ResultMessageSuccess) resultMessage() {}

// ResultMessageError represents an error during query execution.
// This message indicates that an error occurred and contains error details.
type ResultMessageError struct {
	// SessionID identifies the conversation session
	SessionID string

	// ErrorType categorizes the error (e.g., "connection_error", "timeout")
	ErrorType string

	// ErrorMessage provides a human-readable error description
	ErrorMessage string

	// ErrorDetails contains additional error context
	ErrorDetails map[string]any

	// PartialMessages contains any messages received before the error
	PartialMessages []Message

	// PartialUsage contains usage statistics up to the point of error
	PartialUsage *UsageStats
}

func (*ResultMessageError) message()       {}
func (*ResultMessageError) resultMessage() {}
