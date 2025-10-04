package messages

// ResultMessageSuccess indicates a successful query completion.
// This contains the final result along with usage statistics
// and cost information.
type ResultMessageSuccess struct {
	// Subtype is always "success"
	Subtype string `json:"subtype"`

	// DurationMs is total execution time in milliseconds
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is API call time in milliseconds
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is always false for success
	IsError bool `json:"is_error"`

	// NumTurns is the number of conversation turns executed
	NumTurns int `json:"num_turns"`

	// SessionID identifies the conversation session
	SessionID string `json:"session_id"`

	// Result contains the final response text
	Result string `json:"result"`

	// TotalCostUSD is the total cost in US dollars
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregate token usage statistics
	Usage UsageStats `json:"usage"`

	// ModelUsage breaks down usage by model
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tools that were denied by permissions
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

func (ResultMessageSuccess) resultMessage() {}
func (ResultMessageSuccess) message()       {}

// MessageErrorSubtype indicates the type of error that occurred.
type MessageErrorSubtype string

//revive:disable:line-length-limit Long const names required by API
const (
	// MessageErrorSubtypeErrorMaxTurns indicates max turns was reached.
	MessageErrorSubtypeErrorMaxTurns MessageErrorSubtype = "error_max_turns"

	// MessageErrorSubtypeErrorDuringExecution indicates an execution error.
	MessageErrorSubtypeErrorDuringExecution MessageErrorSubtype = "error_during_execution"
)

//revive:enable:line-length-limit

// ResultMessageError indicates an error during execution.
// This contains error information along with usage statistics.
type ResultMessageError struct {
	// Subtype indicates the error type
	Subtype MessageErrorSubtype `json:"subtype"`

	// DurationMs is total execution time in milliseconds
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is API call time in milliseconds
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is always true for errors
	IsError bool `json:"is_error"`

	// NumTurns is the number of conversation turns before error
	NumTurns int `json:"num_turns"`

	// SessionID identifies the conversation session
	SessionID string `json:"session_id"`

	// TotalCostUSD is the total cost in US dollars
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregate token usage statistics
	Usage UsageStats `json:"usage"`

	// ModelUsage breaks down usage by model
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tools that were denied by permissions
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

func (ResultMessageError) resultMessage() {}
func (ResultMessageError) message()       {}

// PermissionDenial represents a tool use that was denied by permissions.
type PermissionDenial struct {
	// ToolName is the name of the denied tool
	ToolName string `json:"tool_name"`

	// ToolUseID is the ID of the denied tool use
	ToolUseID string `json:"tool_use_id"`

	// ToolInput is the input that was attempted (varies by tool)
	ToolInput map[string]any `json:"tool_input"`
}
