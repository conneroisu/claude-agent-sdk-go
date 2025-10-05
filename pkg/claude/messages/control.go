package messages

// ControlRequest wraps SDK to CLI control requests.
// Control requests enable bidirectional communication for operations like
// interruption, permission mode changes, and initialization.
type ControlRequest struct {
	// Type is always "control_request"
	Type string `json:"type"`

	// RequestID uniquely identifies this request for response correlation
	RequestID string `json:"request_id"`

	// Request contains the discriminated request payload
	Request any `json:"request"`
}

// InterruptRequest stops ongoing execution.
type InterruptRequest struct {
	Subtype string `json:"subtype"` // "interrupt"
}

// SetPermissionModeRequest changes the permission mode.
type SetPermissionModeRequest struct {
	Subtype string `json:"subtype"` // "set_permission_mode"
	Mode    string `json:"mode"`    // e.g., "default", "accept_edits"
}

// SetModelRequest switches the AI model.
type SetModelRequest struct {
	Subtype string `json:"subtype"` // "set_model"
	Model   string `json:"model"`   // Model identifier
}

// InitializeRequest sends hook configurations.
type InitializeRequest struct {
	Subtype       string            `json:"subtype"`        // "initialize"
	Version       string            `json:"version"`        // SDK version
	HookCallbacks map[string]string `json:"hook_callbacks"` // callback_id -> hook_name
}

// InboundControlRequest wraps CLI to SDK control requests.
type InboundControlRequest struct {
	// Type is always "control_request"
	Type string `json:"type"`

	// RequestID uniquely identifies this request
	RequestID string `json:"request_id"`

	// Request contains the discriminated request payload
	Request any `json:"request"`
}

// CanUseToolRequest requests permission to use a tool.
type CanUseToolRequest struct {
	Subtype               string              `json:"subtype"` // "can_use_tool"
	ToolName              string              `json:"tool_name"`
	Input                 map[string]any      `json:"input"`
	PermissionSuggestions []PermissionUpdate  `json:"permission_suggestions,omitempty"`
	BlockedPath           *string             `json:"blocked_path,omitempty"`
}

// HookCallbackRequest executes a registered hook.
type HookCallbackRequest struct {
	Subtype    string         `json:"subtype"` // "hook_callback"
	CallbackID string         `json:"callback_id"`
	HookName   string         `json:"hook_name"`
	Input      map[string]any `json:"input"`
}

// MCPMessageRequest routes MCP JSON-RPC messages.
type MCPMessageRequest struct {
	Subtype    string `json:"subtype"` // "mcp_message"
	ServerName string `json:"server_name"`
	Message    []byte `json:"message"` // JSON-RPC message
}

// ControlResponse wraps control request responses.
type ControlResponse struct {
	// Type is always "control_response"
	Type string `json:"type"`

	// RequestID correlates with the original request
	RequestID string `json:"request_id"`

	// Response contains the discriminated response payload
	Response any `json:"response"`
}

// ResponseSuccess represents a successful control response.
type ResponseSuccess struct {
	Success bool           `json:"success"` // Always true
	Result  map[string]any `json:"result,omitempty"`
}

// ResponseError represents an error control response.
type ResponseError struct {
	Success bool   `json:"success"` // Always false
	Error   string `json:"error"`
}

// ControlCancelRequest cancels a pending control request.
type ControlCancelRequest struct {
	// Type is always "control_cancel"
	Type string `json:"type"`

	// RequestID identifies the request to cancel
	RequestID string `json:"request_id"`
}

// PermissionUpdate represents a rule update for tool permissions.
type PermissionUpdate struct {
	// ToolName is the tool to update permissions for
	ToolName string `json:"tool_name"`

	// Rule is the permission rule value
	Rule PermissionRuleValue `json:"rule"`
}

// PermissionRuleValue represents tool-specific permission rules.
type PermissionRuleValue struct {
	// Behavior is the permission action (allow, deny, ask)
	Behavior PermissionBehavior `json:"behavior"`

	// Matcher is an optional pattern for conditional permissions
	Matcher *string `json:"matcher,omitempty"`
}

// PermissionBehavior defines the permission action.
type PermissionBehavior string

const (
	// PermissionAllow always allows the tool
	PermissionAllow PermissionBehavior = "allow"

	// PermissionDeny always denies the tool
	PermissionDeny PermissionBehavior = "deny"

	// PermissionAsk prompts for permission
	PermissionAsk PermissionBehavior = "ask"
)

// PermissionResult discriminates allow/deny decisions.
type PermissionResult interface {
	permissionResult()
}

// PermissionAllowResult allows a tool use.
type PermissionAllowResult struct {
	Allowed bool `json:"allowed"` // Always true
}

func (*PermissionAllowResult) permissionResult() {}

// PermissionDenyResult denies a tool use.
type PermissionDenyResult struct {
	Allowed bool   `json:"allowed"` // Always false
	Reason  string `json:"reason"`
}

func (*PermissionDenyResult) permissionResult() {}
