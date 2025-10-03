// Result message types for Claude Agent.
package messages

// ResultMessage is a discriminated union for query results.
//
// Indicates whether a query completed successfully or with an error.
// Check the Subtype field or use type assertion to determine the variant.
type ResultMessage interface {
	resultMessage()
}

// ResultMessageSuccess indicates a successful query completion.
//
// Contains the final result text, usage statistics, cost information,
// and any permission denials that occurred during execution.
type ResultMessageSuccess struct {
	Subtype           string                `json:"subtype"` // "success"
	DurationMs        int                   `json:"duration_ms"`
	DurationAPIMs     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	SessionID         string                `json:"session_id"`
	Result            string                `json:"result"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             UsageStats            `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
}

func (ResultMessageSuccess) resultMessage() {}
func (ResultMessageSuccess) message()       {}

// MessageErrorSubtype defines error subtypes for ResultMessageError.
type MessageErrorSubtype string

const (
	// MessageErrorSubtypeErrorMaxTurns indicates max turns was reached.
	MessageErrorSubtypeErrorMaxTurns MessageErrorSubtype = "error_max_turns"
	// MessageErrorSubtypeErrorDuringExecution indicates execution failed.
	MessageErrorSubtypeErrorDuringExecution MessageErrorSubtype = "error_during_execution"
)

// ResultMessageError indicates an error during execution.
//
// Contains error details, usage statistics, and information about
// what went wrong during query execution.
type ResultMessageError struct {
	Subtype           MessageErrorSubtype   `json:"subtype"`
	DurationMs        int                   `json:"duration_ms"`
	DurationAPIMs     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	SessionID         string                `json:"session_id"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             UsageStats            `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
}

func (ResultMessageError) resultMessage() {}
func (ResultMessageError) message()       {}

// PermissionDenial represents a tool use denied by permissions.
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"` // Varies by tool
}
