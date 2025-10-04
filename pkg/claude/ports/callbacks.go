package ports

import "context"

// Callback types needed for protocol.go
// These define the hook callback signature used throughout the SDK.

// HookCallback is a function type for hook implementations.
//
// Hook callbacks are invoked at specific points during Claude's execution
// to allow custom logic, validation, or data transformation.
//
// The input parameter contains hook-specific data as a flexible map.
// The context can be used for cancellation and timeout control.
//
// Returns a map containing the hook result, or an error if the hook fails.
//
// Example:
//
//	callback := func(ctx context.Context, input map[string]any) (map[string]any, error) {
//	    toolName := input["tool_name"].(string)
//	    log.Printf("Hook called for tool: %s", toolName)
//	    return map[string]any{"modified": true}, nil
//	}
type HookCallback func(ctx context.Context, input map[string]any) (map[string]any, error)
