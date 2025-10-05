// Package testutil provides test utilities and mocks for testing.
package testutil

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MockPermissionsService implements ports.PermissionsService.
// Provides mock permission checking for testing.
type MockPermissionsService struct {
	CanUseToolFunc CanUseToolFunc
}

// CanUseToolFunc is the function signature for tool permission checks.
type CanUseToolFunc func(
	context.Context,
	string,
	map[string]any,
) (bool, string, error)

// CanUseTool checks if a tool can be used with the given input.
func (m *MockPermissionsService) CanUseTool(
	ctx context.Context,
	toolName string,
	input map[string]any,
) (bool, string, error) {
	if m.CanUseToolFunc != nil {
		return m.CanUseToolFunc(ctx, toolName, input)
	}

	return true, "", nil
}

var _ ports.PermissionsService = (*MockPermissionsService)(nil)
