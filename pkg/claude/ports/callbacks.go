package ports

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// HookCallback is called when a hook event occurs.
// Returns modified input or error.
type HookCallback func(
	ctx context.Context,
	event string,
	input map[string]any,
	toolUseID *string,
) (map[string]any, error)

// PermissionCallback is called to check tool use permission.
// Returns permission decision or error.
type PermissionCallback func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	suggestions []messages.PermissionUpdate,
) (messages.PermissionResult, error)
