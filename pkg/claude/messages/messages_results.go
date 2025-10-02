package messages

// ResultMessage is a discriminated union based on Subtype.
type ResultMessage interface {
	resultMessage()
}

// ResultMessageSuccess indicates a successful query completion.
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

// ResultMessageError indicates an error during execution.
type ResultMessageError struct {
	// "error_max_turns" | "error_during_execution"
	Subtype           string                `json:"subtype"`
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
