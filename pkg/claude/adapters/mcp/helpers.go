package mcp

import (
	"encoding/json"
)

// createErrorResponse creates a JSON-RPC error response.
func createErrorResponse(req map[string]any, code int, message string) ([]byte, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      req["id"],
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	return json.Marshal(response)
}

// createSuccessResponse creates a JSON-RPC success response.
func createSuccessResponse(req map[string]any, result any) ([]byte, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      req["id"],
		"result":  result,
	}
	return json.Marshal(response)
}
