package messages

// UsageStats contains token usage metrics for a conversation.
// Usage statistics track input tokens, output tokens, and cache utilization
// across all models used in the conversation.
type UsageStats struct {
	// InputTokens is the total number of input tokens consumed
	InputTokens int

	// OutputTokens is the total number of output tokens generated
	OutputTokens int

	// CacheCreationInputTokens is tokens used for cache creation
	CacheCreationInputTokens int

	// CacheReadInputTokens is tokens read from cache
	CacheReadInputTokens int

	// PerModel contains per-model usage breakdown
	PerModel map[string]ModelUsage
}

// ModelUsage contains usage statistics for a specific model.
// This allows tracking usage when multiple models are used in a conversation.
type ModelUsage struct {
	// InputTokens is input tokens for this model
	InputTokens int

	// OutputTokens is output tokens for this model
	OutputTokens int

	// CacheCreationInputTokens is cache creation tokens for this model
	CacheCreationInputTokens int

	// CacheReadInputTokens is cache read tokens for this model
	CacheReadInputTokens int
}

// PermissionDenial records a tool use that was denied permission.
// Permission denials track which tools were blocked and why.
type PermissionDenial struct {
	// ToolName is the name of the tool that was denied
	ToolName string

	// Reason describes why permission was denied
	Reason string

	// RequestedInput contains the input that was attempted
	RequestedInput map[string]any
}

// MCPServerStatus represents the connection state of an MCP server.
// Server statuses track whether MCP servers are connected and operational.
type MCPServerStatus struct {
	// Name is the identifier of the MCP server
	Name string

	// Connected indicates whether the server is currently connected
	Connected bool

	// ErrorMessage contains error details if Connected is false
	ErrorMessage *string
}
