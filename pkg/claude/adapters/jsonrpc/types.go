package jsonrpc

import (
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// CanUseToolRequest represents a can_use_tool control request.
// This message asks for permission to use a specific tool.
type CanUseToolRequest struct {
	ToolName string         `json:"tool_name"`
	Input    map[string]any `json:"input"`
	//nolint:revive // line-length-limit: JSON tag exceeds 80 char limit
	PermissionSuggestions []permissions.PermissionUpdate `json:"permission_suggestions,omitempty"`
}

// CanUseToolResponse represents a can_use_tool control response.
// It indicates whether the tool use is allowed.
type CanUseToolResponse struct {
	Allow  bool           `json:"allow"`
	Input  map[string]any `json:"input,omitempty"`
	Reason string         `json:"reason,omitempty"`
	//nolint:revive // line-length-limit: JSON tag exceeds 80 char limit
	UpdatedPermissions []permissions.PermissionUpdate `json:"updated_permissions,omitempty"`
}

// HookCallbackRequest represents a hook_callback control request.
// This message triggers a registered lifecycle hook.
type HookCallbackRequest struct {
	CallbackID string         `json:"callback_id"`
	Input      map[string]any `json:"input"`
	ToolUseID  *string        `json:"tool_use_id,omitempty"`
}

// MCPMessageRequest represents an mcp_message control request.
// This message forwards a JSON-RPC message to an MCP server.
type MCPMessageRequest struct {
	ServerName string         `json:"server_name"`
	Message    map[string]any `json:"message"`
}

// MCPMessageResponse represents an mcp_message control response.
// It contains the JSON-RPC response from the MCP server.
type MCPMessageResponse struct {
	MCPResponse map[string]any `json:"mcp_response"`
}

// PermissionUpdateDTO represents the raw JSON structure for
// permission updates, used for unmarshaling.
type PermissionUpdateDTO struct {
	Type     string                          `json:"type"`
	Rules    []PermissionRuleValueDTO        `json:"rules,omitempty"`
	Behavior *permissions.PermissionBehavior `json:"behavior,omitempty"`
	Mode     *options.PermissionMode         `json:"mode,omitempty"`
	//nolint:revive // line-length-limit: JSON tag exceeds 80 char limit
	Directories []string `json:"directories,omitempty"`
	//nolint:revive // line-length-limit: JSON tag exceeds 80 char limit
	Destination *permissions.PermissionUpdateDestination `json:"destination,omitempty"`
}

// PermissionRuleValueDTO represents the raw JSON structure for
// a permission rule value.
type PermissionRuleValueDTO struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitempty"`
}

// ToPermissionUpdate converts DTO to domain type.
//nolint:revive // receiver-naming: dto is conventional for DTOs
func (dto PermissionUpdateDTO) ToPermissionUpdate() permissions.PermissionUpdate {
	rules := make([]permissions.PermissionRuleValue, 0, len(dto.Rules))
	for _, r := range dto.Rules {
		rules = append(rules, permissions.PermissionRuleValue{
			ToolName:    r.ToolName,
			RuleContent: r.RuleContent,
		})
	}

	return permissions.PermissionUpdate{
		Type:        dto.Type,
		Rules:       rules,
		Behavior:    dto.Behavior,
		Mode:        dto.Mode,
		Directories: dto.Directories,
		Destination: dto.Destination,
	}
}

// ToPermissionUpdates converts a slice of DTOs to domain types.
//nolint:lll // function name and signature exceed line limit
func ToPermissionUpdates(
	dtos []PermissionUpdateDTO,
) []permissions.PermissionUpdate {
	updates := make([]permissions.PermissionUpdate, 0, len(dtos))
	for _, dto := range dtos {
		updates = append(updates, dto.ToPermissionUpdate())
	}

	return updates
}
