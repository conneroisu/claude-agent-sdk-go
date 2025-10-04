package messages

// ResultMessage is a discriminated union for query execution results.
//
// Results can be either success or error, distinguished by the Subtype field.
// Both variants share common metrics (duration, cost, usage, etc.).
type ResultMessage interface {
	resultMessage()
}

// ResultMessageSuccess indicates a successful query completion.
//
// Contains the final result text, execution metrics, token usage,
// and cost information. Sent when Claude completes a query successfully.
//
// Example:
//
//	result := ResultMessageSuccess{
//	    Subtype: "success",
//	    Result: "Task completed successfully",
//	    NumTurns: 5,
//	    TotalCostUSD: 0.025,
//	}
type ResultMessageSuccess struct {
	// Subtype is always "success" for successful results.
	Subtype string `json:"subtype"`

	// DurationMs is the total execution time in milliseconds.
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is the time spent in API calls in milliseconds.
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is always false for success results.
	IsError bool `json:"is_error"`

	// NumTurns is the number of conversation turns executed.
	NumTurns int `json:"num_turns"`

	// SessionID identifies the conversation session.
	SessionID string `json:"session_id"`

	// Result is the final result text from Claude.
	Result string `json:"result"`

	// TotalCostUSD is the estimated cost in US dollars.
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregated token usage statistics.
	Usage UsageStats `json:"usage"`

	// ModelUsage maps model names to their individual usage stats.
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tools that were denied by permissions.
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

// resultMessage implements the ResultMessage interface.
func (ResultMessageSuccess) resultMessage() {}

// message implements the Message interface.
func (ResultMessageSuccess) message() {}

// MessageErrorSubtype represents the type of error that occurred.
type MessageErrorSubtype string

const (
	// MessageErrorSubtypeErrorMaxTurns indicates max turns limit reached.
	MessageErrorSubtypeErrorMaxTurns MessageErrorSubtype = "error_max_turns"

	// MessageErrorSubtypeErrorDuringExecution indicates execution error.
	//nolint:revive // Constant name from API spec
	MessageErrorSubtypeErrorDuringExecution MessageErrorSubtype = "error_during_execution"
)

// ResultMessageError indicates an error during query execution.
//
// Contains execution metrics and error information. Sent when Claude
// encounters an error or reaches the maximum turn limit.
//
// Example:
//
//	result := ResultMessageError{
//	    Subtype: MessageErrorSubtypeErrorMaxTurns,
//	    NumTurns: 25,
//	    IsError: true,
//	}
type ResultMessageError struct {
	// Subtype indicates the type of error.
	// Values: "error_max_turns", "error_during_execution"
	Subtype MessageErrorSubtype `json:"subtype"`

	// DurationMs is the total execution time in milliseconds.
	DurationMs int `json:"duration_ms"`

	// DurationAPIMs is the time spent in API calls in milliseconds.
	DurationAPIMs int `json:"duration_api_ms"`

	// IsError is always true for error results.
	IsError bool `json:"is_error"`

	// NumTurns is the number of conversation turns executed before error.
	NumTurns int `json:"num_turns"`

	// SessionID identifies the conversation session.
	SessionID string `json:"session_id"`

	// TotalCostUSD is the estimated cost in US dollars.
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains aggregated token usage statistics.
	Usage UsageStats `json:"usage"`

	// ModelUsage maps model names to their individual usage stats.
	ModelUsage map[string]ModelUsage `json:"modelUsage"`

	// PermissionDenials lists tools that were denied by permissions.
	PermissionDenials []PermissionDenial `json:"permission_denials"`
}

// resultMessage implements the ResultMessage interface.
func (ResultMessageError) resultMessage() {}

// message implements the Message interface.
func (ResultMessageError) message() {}

// PermissionDenial represents a tool use that was denied by permissions.
//
// Contains the tool name, tool use ID, and the input that was denied.
// Used to track permission-based rejections in query results.
//
// Example:
//
//	denial := PermissionDenial{
//	    ToolName: "Bash",
//	    ToolUseID: "toolu_123",
//	    ToolInput: map[string]any{"command": "rm -rf /"},
//	}
type PermissionDenial struct {
	// ToolName is the name of the tool that was denied.
	ToolName string `json:"tool_name"`

	// ToolUseID is the ID of the denied tool use.
	ToolUseID string `json:"tool_use_id"`

	// ToolInput contains the input parameters that were denied.
	// Flexible map since tool inputs vary by tool.
	ToolInput map[string]any `json:"tool_input"`
}
