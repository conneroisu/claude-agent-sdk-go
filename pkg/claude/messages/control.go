package messages

// ControlMessage types for bidirectional communication with Claude CLI.
// Based on TypeScript SDK control protocol specification.
//
// Control Protocol Flow:
//  1. SDK → CLI: Send ControlRequest with unique request_id
//  2. CLI → SDK: Respond with ControlResponse matching the request_id
//  3. CLI → SDK: Send InboundControlRequest for permissions/hooks/MCP
//  4. SDK → CLI: Respond with ControlResponse for the inbound request
//
// Request ID Format: req_{counter}_{randomHex(4)}
// Example: "req_1_a3f2", "req_2_b4c1"

// ControlRequest is sent from SDK to CLI.
//
// Contains a unique request ID and one of several request types:
// InterruptRequest, SetPermissionModeRequest, SetModelRequest,
// or InitializeRequest.
//
// Example:
//
//	req := ControlRequest{
//	    Type: "control_request",
//	    RequestID: "req_1_a3f2",
//	    Request: SetModelRequest{
//	        Subtype: "set_model",
//	        Model: StringPtr("claude-opus-4-20250514"),
//	    },
//	}
type ControlRequest struct {
	// Type is always "control_request" for outbound requests.
	Type string `json:"type"`

	// RequestID uniquely identifies this request for response matching.
	// Format: req_{counter}_{randomHex(4)}
	RequestID string `json:"request_id"`

	// Request contains the specific request payload.
	// One of: InterruptRequest, SetPermissionModeRequest,
	// SetModelRequest, InitializeRequest
	Request any `json:"request"`
}

// ControlResponse is received from CLI in response to SDK requests.
//
// Contains the response payload which includes the matching request ID
// and either a success response or error message.
//
// Example:
//
//	resp := ControlResponse{
//	    Type: "control_response",
//	    Response: ResponseUnion{
//	        Subtype: "success",
//	        RequestID: "req_1_a3f2",
//	        Response: map[string]any{"status": "ok"},
//	    },
//	}
type ControlResponse struct {
	// Type is always "control_response" for responses.
	Type string `json:"type"`

	// Response contains the response payload with result or error.
	Response ResponseUnion `json:"response"`
}

// ResponseUnion represents a success or error response.
//
// Distinguished by the Subtype field ("success" or "error").
// Success responses include the Response map,
// error responses include an Error message.
//
// Example success:
//
//	union := ResponseUnion{
//	    Subtype: "success",
//	    RequestID: "req_1_a3f2",
//	    Response: map[string]any{"updated": true},
//	}
//
// Example error:
//
//	union := ResponseUnion{
//	    Subtype: "error",
//	    RequestID: "req_1_a3f2",
//	    Error: "Invalid model name",
//	}
type ResponseUnion struct {
	// Subtype indicates success or error.
	// Values: "success", "error"
	Subtype string `json:"subtype"`

	// RequestID matches the request this responds to.
	RequestID string `json:"request_id"`

	// Response contains the success response data.
	// Present when Subtype is "success".
	Response map[string]any `json:"response,omitempty"`

	// Error contains the error message.
	// Present when Subtype is "error".
	Error string `json:"error,omitempty"`
}

// SDK → CLI Control Requests

// InterruptRequest requests interruption of the current execution.
//
// Sent when the SDK wants to stop Claude's current query execution.
//
// Example:
//
//	req := InterruptRequest{Subtype: "interrupt"}
type InterruptRequest struct {
	// Subtype is always "interrupt".
	Subtype string `json:"subtype"`
}

// SetPermissionModeRequest changes the permission mode.
//
// Sent to dynamically change how permissions are handled during execution.
//
// Example:
//
//	req := SetPermissionModeRequest{
//	    Subtype: "set_permission_mode",
//	    Mode: "acceptEdits",
//	}
type SetPermissionModeRequest struct {
	// Subtype is always "set_permission_mode".
	Subtype string `json:"subtype"`

	// Mode is the new permission mode.
	// Values: "default", "acceptEdits", "plan", "bypassPermissions", "ask"
	Mode string `json:"mode"`
}

// SetModelRequest changes the active model.
//
// Sent to switch to a different Claude model during execution.
// Set Model to nil to reset to default.
//
// Example:
//
//	req := SetModelRequest{
//	    Subtype: "set_model",
//	    Model: StringPtr("claude-opus-4-20250514"),
//	}
type SetModelRequest struct {
	// Subtype is always "set_model".
	Subtype string `json:"subtype"`

	// Model is the new model name, or nil to reset to default.
	Model *string `json:"model"`
}

// InitializeRequest initializes the session with hook configurations.
//
// Sent at session start to register SDK hooks with the CLI.
//
// Example:
//
//	req := InitializeRequest{
//	    Subtype: "initialize",
//	    Hooks: map[string]any{
//	        "beforeToolUse": map[string]any{"enabled": true},
//	    },
//	}
type InitializeRequest struct {
	// Subtype is always "initialize".
	Subtype string `json:"subtype"`

	// Hooks contains hook configurations.
	// Maps hook names to their configuration objects.
	Hooks map[string]any `json:"hooks"`
}

// CLI → SDK Control Requests (inbound)

// InboundControlRequest is sent from CLI to SDK.
//
// Requests the SDK to perform an action or make a decision.
// Request types: CanUseToolRequest, HookCallbackRequest, MCPMessageRequest.
//
// Example:
//
//	req := InboundControlRequest{
//	    Type: "control_request",
//	    RequestID: "req_cli_123",
//	    Request: CanUseToolRequest{
//	        Subtype: "can_use_tool",
//	        ToolName: "Bash",
//	        Input: map[string]any{"command": "ls"},
//	    },
//	}
type InboundControlRequest struct {
	// Type is always "control_request" for inbound requests.
	Type string `json:"type"`

	// RequestID uniquely identifies this request for response matching.
	RequestID string `json:"request_id"`

	// Request contains the specific request payload.
	// One of: CanUseToolRequest, HookCallbackRequest, MCPMessageRequest
	Request any `json:"request"`
}

// CanUseToolRequest asks if a tool use is permitted.
//
// Sent by CLI to check permissions before executing a tool.
// The SDK responds with PermissionResultAllow or PermissionResultDeny.
//
// Example:
//
//	req := CanUseToolRequest{
//	    Subtype: "can_use_tool",
//	    ToolName: "Bash",
//	    Input: map[string]any{"command": "git status"},
//	}
type CanUseToolRequest struct {
	// Subtype is always "can_use_tool".
	Subtype string `json:"subtype"`

	// ToolName is the name of the tool to check.
	ToolName string `json:"tool_name"`

	// Input contains the tool input parameters to check.
	Input map[string]any `json:"input"`

	// PermissionSuggestions contains suggested permission updates.
	//nolint:revive // Line length from API spec
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`

	// BlockedPath contains the path that was blocked, if applicable.
	BlockedPath *string `json:"blocked_path,omitempty"`
}

// HookCallbackRequest invokes a registered hook callback.
//
// Sent by CLI when a hook point is reached during execution.
// The SDK executes the registered callback and returns the result.
//
// Example:
//
//	req := HookCallbackRequest{
//	    Subtype: "hook_callback",
//	    CallbackID: "beforeToolUse_123",
//	    Input: map[string]any{
//	        "tool_name": "Bash",
//	        "input": map[string]any{"command": "ls"},
//	    },
//	}
type HookCallbackRequest struct {
	// Subtype is always "hook_callback".
	Subtype string `json:"subtype"`

	// CallbackID identifies which hook to execute.
	CallbackID string `json:"callback_id"`

	// Input contains hook-specific input data.
	Input map[string]any `json:"input"`

	// ToolUseID identifies the tool use, if applicable.
	ToolUseID *string `json:"tool_use_id,omitempty"`
}

// MCPMessageRequest routes a message to an MCP server.
//
// Sent by CLI to forward JSON-RPC messages to SDK-managed MCP servers.
// The SDK routes the message and returns the server's response.
//
// Example:
//
//	req := MCPMessageRequest{
//	    Subtype: "mcp_message",
//	    ServerName: "filesystem",
//	    Message: map[string]any{
//	        "jsonrpc": "2.0",
//	        "method": "tools/list",
//	        "id": 1,
//	    },
//	}
type MCPMessageRequest struct {
	// Subtype is always "mcp_message".
	Subtype string `json:"subtype"`

	// ServerName identifies which MCP server to route to.
	ServerName string `json:"server_name"`

	// Message contains the raw JSON-RPC message to route.
	Message map[string]any `json:"message"`
}

// ControlCancelRequest cancels a pending control request.
//
// Sent to abort a control request that hasn't completed yet.
//
// Example:
//
//	cancel := ControlCancelRequest{
//	    Type: "control_cancel_request",
//	    RequestID: "req_1_a3f2",
//	}
type ControlCancelRequest struct {
	// Type is always "control_cancel_request".
	Type string `json:"type"`

	// RequestID identifies the request to cancel.
	RequestID string `json:"request_id"`
}

// Permission Types (used in control protocol)

// PermissionUpdate represents a permission configuration change.
//
// Used to add, remove, or modify permission rules and settings.
// The Type field determines which fields are relevant.
//
// Example:
//
//	update := PermissionUpdate{
//	    Type: "addRules",
//	    Rules: []PermissionRuleValue{
//	        {ToolName: "Bash", RuleContent: StringPtr("git:*")},
//	    },
//	    Behavior: PermissionBehaviorPtr(PermissionBehaviorAllow),
//	}
type PermissionUpdate struct {
	// Type indicates the update operation.
	// Values: "addRules", "replaceRules", "removeRules", "setMode",
	//         "addDirectories", "removeDirectories"
	Type string `json:"type"`

	// Rules contains permission rules (for rule-related updates).
	Rules []PermissionRuleValue `json:"rules,omitempty"`

	// Behavior specifies how to handle matching rules.
	Behavior *PermissionBehavior `json:"behavior,omitempty"`

	// Mode sets the permission mode (for "setMode" updates).
	Mode *string `json:"mode,omitempty"`

	// Directories lists directories (for directory-related updates).
	Directories []string `json:"directories,omitempty"`

	// Destination specifies where to save the update.
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// PermissionRuleValue represents a single permission rule.
//
// Defines a rule for a specific tool, optionally with a content matcher.
//
// Example:
//
//	rule := PermissionRuleValue{
//	    ToolName: "Bash",
//	    RuleContent: StringPtr("git:*"),
//	}
type PermissionRuleValue struct {
	// ToolName is the tool this rule applies to.
	ToolName string `json:"toolName"`

	// RuleContent is an optional matcher pattern.
	// Example: "git:*" to match git commands in Bash.
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionBehavior specifies how to handle a permission check.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow permits the action.
	PermissionBehaviorAllow PermissionBehavior = "allow"

	// PermissionBehaviorDeny rejects the action.
	PermissionBehaviorDeny PermissionBehavior = "deny"

	// PermissionBehaviorAsk prompts the user.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination specifies where to save permission updates.
type PermissionUpdateDestination string

const (
	// PermissionUpdateDestinationUserSettings saves to user-level config.
	//nolint:revive // Constant name from API spec
	PermissionUpdateDestinationUserSettings PermissionUpdateDestination = "userSettings"

	// PermissionUpdateDestinationProjectSettings saves to project config.
	//nolint:revive // Constant name from API spec
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"

	// PermissionUpdateDestinationLocalSettings saves to local config.
	//nolint:revive // Constant name from API spec
	PermissionUpdateDestinationLocalSettings PermissionUpdateDestination = "localSettings"

	// PermissionUpdateDestinationSession saves for this session only.
	PermissionUpdateDestinationSession PermissionUpdateDestination = "session"
)

// Permission Result types (returned by can_use_tool callback)

// PermissionResult represents the result of a permission check.
//
// Can be either PermissionResultAllow or PermissionResultDeny.
type PermissionResult interface {
	permissionResult()
}

// PermissionResultAllow permits a tool use.
//
// Can optionally modify the input or update permission rules.
//
// Example:
//
//	result := PermissionResultAllow{
//	    Behavior: "allow",
//	    UpdatedInput: map[string]any{
//	        "command": "git status --short",
//	    },
//	}
type PermissionResultAllow struct {
	// Behavior is always "allow".
	Behavior string `json:"behavior"`

	// UpdatedInput contains modified tool input, if changed.
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`

	// UpdatedPermissions contains permission updates to apply.
	UpdatedPermissions []PermissionUpdate `json:"updatedPermissions,omitempty"`
}

// permissionResult implements the PermissionResult interface.
func (PermissionResultAllow) permissionResult() {}

// PermissionResultDeny rejects a tool use.
//
// Includes a user-facing message explaining the denial.
// Can optionally interrupt execution completely.
//
// Example:
//
//	result := PermissionResultDeny{
//	    Behavior: "deny",
//	    Message: "Cannot delete system files",
//	    Interrupt: true,
//	}
type PermissionResultDeny struct {
	// Behavior is always "deny".
	Behavior string `json:"behavior"`

	// Message explains why the tool use was denied.
	Message string `json:"message"`

	// Interrupt stops execution completely if true.
	Interrupt bool `json:"interrupt,omitempty"`
}

// permissionResult implements the PermissionResult interface.
func (PermissionResultDeny) permissionResult() {}
