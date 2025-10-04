package messages

// ControlMessage types enable bidirectional communication with Claude CLI
// using JSON-RPC over stdin/stdout. This follows the TypeScript SDK
// control protocol specification.

// ControlRequest is sent from SDK to CLI.
type ControlRequest struct {
	// Type is always "control_request"
	Type string `json:"type"`

	// RequestID uniquely identifies this request for correlation
	RequestID string `json:"request_id"`

	// Request contains one of the control request subtypes
	Request any `json:"request"`
}

// ControlResponse is received from CLI in response to SDK requests.
type ControlResponse struct {
	// Type is always "control_response"
	Type string `json:"type"`

	// Response contains success or error information
	Response ResponseUnion `json:"response"`
}

// ResponseUnion discriminates between success and error responses.
type ResponseUnion struct {
	// Subtype is "success" or "error"
	Subtype string `json:"subtype"`

	// RequestID links this response to the original request
	RequestID string `json:"request_id"`

	// Response is present when subtype is "success"
	Response map[string]any `json:"response,omitempty"`

	// Error is present when subtype is "error"
	Error string `json:"error,omitempty"`
}

// InterruptRequest cancels ongoing operations.
type InterruptRequest struct {
	// Subtype is always "interrupt"
	Subtype string `json:"subtype"`
}

// SetPermissionModeRequest changes the permission handling mode.
type SetPermissionModeRequest struct {
	// Subtype is always "set_permission_mode"
	Subtype string `json:"subtype"`

	// Mode is the new permission mode
	Mode string `json:"mode"`
}

// SetModelRequest changes the AI model.
type SetModelRequest struct {
	// Subtype is always "set_model"
	Subtype string `json:"subtype"`

	// Model is the new model (nil to reset to default)
	Model *string `json:"model"`
}

// InitializeRequest initializes a session with hooks and configuration.
type InitializeRequest struct {
	// Subtype is always "initialize"
	Subtype string `json:"subtype"`

	// Hooks contains hook callback configurations
	Hooks map[string]any `json:"hooks"`
}

// InboundControlRequest is received from CLI requesting SDK action.
type InboundControlRequest struct {
	// Type is always "control_request"
	Type string `json:"type"`

	// RequestID uniquely identifies this request
	RequestID string `json:"request_id"`

	// Request contains one of the inbound request subtypes
	Request any `json:"request"`
}

// CanUseToolRequest asks the SDK for permission to use a tool.
type CanUseToolRequest struct {
	// Subtype is always "can_use_tool"
	Subtype string `json:"subtype"`

	// ToolName is the tool being requested
	ToolName string `json:"tool_name"`

	// Input contains tool-specific parameters
	Input map[string]any `json:"input"`

	// PermissionSuggestions are suggested "always allow" rules
	//revive:disable:line-length-limit JSON tag requires full field name
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
	//revive:enable:line-length-limit

	// BlockedPath indicates a path that triggered a permission check
	BlockedPath *string `json:"blocked_path,omitempty"`
}

// HookCallbackRequest asks the SDK to execute a registered hook.
type HookCallbackRequest struct {
	// Subtype is always "hook_callback"
	Subtype string `json:"subtype"`

	// CallbackID identifies which hook to execute
	CallbackID string `json:"callback_id"`

	// Input contains hook-specific parameters
	Input map[string]any `json:"input"`

	// ToolUseID links this hook to a tool use if applicable
	ToolUseID *string `json:"tool_use_id,omitempty"`
}

// MCPMessageRequest routes an MCP JSON-RPC message.
type MCPMessageRequest struct {
	// Subtype is always "mcp_message"
	Subtype string `json:"subtype"`

	// ServerName identifies which MCP server to route to
	ServerName string `json:"server_name"`

	// Message contains the raw JSON-RPC message
	Message map[string]any `json:"message"`
}

// ControlCancelRequest cancels a pending control request.
type ControlCancelRequest struct {
	// Type is always "control_cancel_request"
	Type string `json:"type"`

	// RequestID identifies which request to cancel
	RequestID string `json:"request_id"`
}

// PermissionUpdate represents a permission configuration change.
type PermissionUpdate struct {
	// Type indicates the update operation
	Type string `json:"type"`

	// Rules contains permission rules (for rule operations)
	Rules []PermissionRuleValue `json:"rules,omitempty"`

	// Behavior is the permission action (for rule operations)
	Behavior *PermissionBehavior `json:"behavior,omitempty"`

	// Mode is the new permission mode (for setMode operation)
	Mode *string `json:"mode,omitempty"`

	// Directories are paths (for directory operations)
	Directories []string `json:"directories,omitempty"`

	// Destination specifies where to save the update
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// PermissionRuleValue defines a permission rule.
type PermissionRuleValue struct {
	// ToolName is the tool this rule applies to
	ToolName string `json:"toolName"`

	// RuleContent is the rule pattern (e.g., "git:*")
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionBehavior defines how a permission rule behaves.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow automatically allows the tool.
	PermissionBehaviorAllow PermissionBehavior = "allow"

	// PermissionBehaviorDeny automatically denies the tool.
	PermissionBehaviorDeny PermissionBehavior = "deny"

	// PermissionBehaviorAsk prompts the user for permission.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where to save permission updates.
type PermissionUpdateDestination string

//revive:disable:line-length-limit Long type and const names required by API
const (
	// PermissionUpdateDestinationUserSettings saves to user settings.
	PermissionUpdateDestinationUserSettings PermissionUpdateDestination = "userSettings"

	// PermissionUpdateDestinationProjectSettings saves to project settings.
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"

	// PermissionUpdateDestinationLocalSettings saves to local settings.
	PermissionUpdateDestinationLocalSettings PermissionUpdateDestination = "localSettings"

	// PermissionUpdateDestinationSession saves for current session only.
	PermissionUpdateDestinationSession PermissionUpdateDestination = "session"
)

//revive:enable:line-length-limit

// PermissionResult is returned by can_use_tool callbacks.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow grants permission to use the tool.
type PermissionResultAllow struct {
	// Behavior is always "allow"
	Behavior string `json:"behavior"`

	// UpdatedInput can modify the tool input
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`

	// UpdatedPermissions can update permission rules
	UpdatedPermissions []PermissionUpdate `json:"updatedPermissions,omitempty"`
}

func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny denies permission to use the tool.
type PermissionResultDeny struct {
	// Behavior is always "deny"
	Behavior string `json:"behavior"`

	// Message explains why permission was denied
	Message string `json:"message"`

	// Interrupt stops execution completely if true
	Interrupt bool `json:"interrupt,omitempty"`
}

func (PermissionResultDeny) permissionResult() {}
