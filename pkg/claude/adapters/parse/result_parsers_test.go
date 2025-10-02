// Package parse provides message parsing adapters for the Claude SDK.
// This test file validates the type-safe JSON-based result message parsing.
package parse

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

func TestParseResultMessageV2_Success(t *testing.T) {
	data := map[string]any{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     1234.0,
		"duration_api_ms": 567.0,
		"is_error":        false,
		"num_turns":       2.0,
		"session_id":      "test-session-123",
		"result":          "Task completed successfully",
		"total_cost_usd":  0.001,
		"usage": map[string]any{
			"input_tokens":                1000.0,
			"output_tokens":               500.0,
			"cache_read_input_tokens":     100.0,
			"cache_creation_input_tokens": 50.0,
		},
		"modelUsage": map[string]any{
			"claude-sonnet-4": map[string]any{
				"inputTokens":              1000.0,
				"outputTokens":             500.0,
				"cacheReadInputTokens":     100.0,
				"cacheCreationInputTokens": 50.0,
				"webSearchRequests":        0.0,
				"costUSD":                  0.001,
				"contextWindow":            200000.0,
			},
		},
		"permission_denials": []any{},
	}

	msg, err := parseResultMessageV2(data)
	if err != nil {
		t.Fatalf("Failed to parse success result: %v", err)
	}

	successMsg, ok := msg.(*messages.ResultMessageSuccess)
	if !ok {
		t.Fatalf(
			"Expected *ResultMessageSuccess, got %T",
			msg,
		)
	}

	// Verify fields were correctly parsed
	if successMsg.DurationMs != 1234 {
		t.Errorf(
			"Expected DurationMs=1234, got %d",
			successMsg.DurationMs,
		)
	}
	if successMsg.DurationAPIMs != 567 {
		t.Errorf(
			"Expected DurationAPIMs=567, got %d",
			successMsg.DurationAPIMs,
		)
	}
	if successMsg.IsError {
		t.Error("Expected IsError=false")
	}
	if successMsg.NumTurns != 2 {
		t.Errorf("Expected NumTurns=2, got %d", successMsg.NumTurns)
	}
	if successMsg.SessionID != "test-session-123" {
		t.Errorf(
			"Expected SessionID='test-session-123', got %s",
			successMsg.SessionID,
		)
	}
	if successMsg.Result != "Task completed successfully" {
		t.Errorf(
			"Expected Result='Task completed successfully', got %s",
			successMsg.Result,
		)
	}
}

func TestParseResultMessageV2_Error(t *testing.T) {
	data := map[string]any{
		"type":            "result",
		"subtype":         "error_max_turns",
		"duration_ms":     5000.0,
		"duration_api_ms": 4500.0,
		"is_error":        true,
		"num_turns":       10.0,
		"session_id":      "test-session-456",
		"total_cost_usd":  0.05,
		"usage": map[string]any{
			"input_tokens":                10000.0,
			"output_tokens":               5000.0,
			"cache_read_input_tokens":     0.0,
			"cache_creation_input_tokens": 0.0,
		},
		"modelUsage":         map[string]any{},
		"permission_denials": []any{},
	}

	msg, err := parseResultMessageV2(data)
	if err != nil {
		t.Fatalf("Failed to parse error result: %v", err)
	}

	errorMsg, ok := msg.(*messages.ResultMessageError)
	if !ok {
		t.Fatalf("Expected *ResultMessageError, got %T", msg)
	}

	// Verify fields were correctly parsed
	if errorMsg.DurationMs != 5000 {
		t.Errorf(
			"Expected DurationMs=5000, got %d",
			errorMsg.DurationMs,
		)
	}
	if errorMsg.Subtype != "error_max_turns" {
		t.Errorf(
			"Expected Subtype='error_max_turns', got %s",
			errorMsg.Subtype,
		)
	}
	if !errorMsg.IsError {
		t.Error("Expected IsError=true")
	}
}

func TestParseResultMessageV2_InvalidSubtype(t *testing.T) {
	data := map[string]any{
		"type":    "result",
		"subtype": "invalid_subtype",
	}

	_, err := parseResultMessageV2(data)
	if err == nil {
		t.Fatal("Expected error for invalid subtype, got nil")
	}
}

func TestParseResultMessageV2_MissingSubtype(t *testing.T) {
	data := map[string]any{
		"type": "result",
	}

	_, err := parseResultMessageV2(data)
	if err == nil {
		t.Fatal("Expected error for missing subtype, got nil")
	}
}
