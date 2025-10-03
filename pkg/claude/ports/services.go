// Service interfaces for dependency injection in protocol handler.
//
// These interfaces allow the protocol handler to coordinate with
// domain services without creating circular dependencies.
package ports

import "context"

// PermissionService defines what the protocol handler needs
// from the permissions service.
//
// The permissions service checks whether a tool use is allowed
// based on permission mode and user-defined callbacks.
type PermissionService interface {
	// CheckToolUse verifies if a tool can be used.
	CheckToolUse(
		ctx context.Context,
		toolName string,
		input map[string]any,
		suggestions any,
	) (any, error)
}

// HookCallback is a user-defined function for lifecycle hooks.
//
// Hook callbacks are invoked at various points in the agent lifecycle
// (e.g., before/after tool use, on errors, etc.).
// The input is hook-specific data, and the return value is sent back.
type HookCallback func(
	input map[string]any,
	toolUseID *string,
	ctx any,
) (map[string]any, error)
