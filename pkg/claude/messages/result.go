package messages

// ResultMessageSuccess indicates successful query completion.
// Implements both ResultMessage and Message interfaces.
type ResultMessageSuccess struct {
	// Subtype is always "success"
	Subtype string `json:"subtype"`

	// DurationMs is total execution time in milliseconds
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is API call time in milliseconds
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is false for success messages
	IsError bool `json:"is_error"`

	// NumTurns is count of conversation turns
	NumTurns int `json:"num_turns"`

	// SessionID identifies this conversation session
	SessionID string `json:"session_id"`

	// Result contains the final response text
	Result string `json:"result"`

	// TotalCostUSD is total API cost in US dollars
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregated token usage statistics
	Usage UsageStats `json:"usage"`

	// ModelUsage maps model names to their usage statistics
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tool uses denied by permissions
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

// resultMessage implements the ResultMessage interface.
func (ResultMessageSuccess) resultMessage() {}

// message implements the Message interface.
func (ResultMessageSuccess) message() {}

// MessageErrorSubtype identifies error result variants.
type MessageErrorSubtype string

const (
	// MessageErrorSubtypeErrorMaxTurns indicates max turns
	// was reached.
//nolint:revive,lll // line-length-limit: constant name clarity

	// MessageErrorSubtypeErrorDuringExecution indicates an
	// error occurred during execution.
//nolint:revive,lll // line-length-limit: constant name clarity
)

// ResultMessageError indicates an error during execution.
// Implements both ResultMessage and Message interfaces.
type ResultMessageError struct {
	// Subtype identifies the error type
	Subtype MessageErrorSubtype `json:"subtype"`

	// DurationMs is total execution time in milliseconds
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is API call time in milliseconds
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is true for error messages
	IsError bool `json:"is_error"`

	// NumTurns is count of conversation turns before error
	NumTurns int `json:"num_turns"`

	// SessionID identifies this conversation session
	SessionID string `json:"session_id"`

	// TotalCostUSD is total API cost in US dollars
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregated token usage statistics
	Usage UsageStats `json:"usage"`

	// ModelUsage maps model names to their usage statistics
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tool uses denied by permissions
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

// resultMessage implements the ResultMessage interface.
func (ResultMessageError) resultMessage() {}

// message implements the Message interface.
func (ResultMessageError) message() {}

// PermissionDenial represents a tool use denied by permission
// checks.
type PermissionDenial struct {
	// ToolName is the name of the denied tool
	ToolName string `json:"tool_name"`

	// ToolUseID identifies the specific tool use attempt
	ToolUseID string `json:"tool_use_id"`

	// ToolInput contains the tool's input parameters.
	// Intentionally flexible as inputs vary by tool.
	ToolInput map[string]any `json:"tool_input"`
}
