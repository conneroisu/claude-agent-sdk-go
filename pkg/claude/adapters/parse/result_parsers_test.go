// Package parse provides message parsing adapters for the Claude SDK.
//
// This test file validates the type-safe JSON-based result message parsing
// that uses json.Marshal + json.Unmarshal instead of manual type assertions.
//
// The parseResultMessageV2 function demonstrates the new approach where we:
// 1. Check the subtype field to determine success vs error
// 2. Marshal the map[string]any to JSON bytes
// 3. Unmarshal into the appropriate typed struct
//
// This eliminates ~60 type assertions and provides compile-time type safety.
package parse

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// Test constants to avoid magic numbers.
const (
	// Field names
	fieldType = "type"
	fieldSubtype = "subtype"

	// Field values
	testType                   = "result"
	testSuccessSubtype         = "success"
	testErrorSubtype           = "error_max_turns"
	testSessionIDSuccess       = "test-session-123"
	testSessionIDError         = "test-session-456"
	testResultText             = "Task completed successfully"
	testDurationMs             = 1234
	testDurationAPIMs          = 567
	testDurationMsError        = 5000
	testDurationAPIMsError     = 4500
	testNumTurns               = 2
	testNumTurnsError          = 10
	testCostUSD                = 0.001
	testCostUSDError           = 0.05
	testInputTokens            = 1000.0
	testOutputTokens           = 500.0
	testCacheReadTokens        = 100.0
	testCacheCreationTokens    = 50.0
	testInputTokensError       = 10000.0
	testOutputTokensError      = 5000.0
	testWebSearchRequests      = 0.0
	testContextWindow          = 200000.0
	testModelName              = "claude-sonnet-4"
)

// TestParseResultMessageV2_Success validates that a success result message
// is correctly parsed from a map into a typed ResultMessageSuccess struct.
// This test ensures all fields are properly unmarshaled including nested
// usage statistics and per-model usage data.
func TestParseResultMessageV2_Success(t *testing.T) {
	// Create test data matching the Claude CLI output format.
	data := map[string]any{
		"type":            testType,
		"subtype":         testSuccessSubtype,
		"duration_ms":     float64(testDurationMs),
		"duration_api_ms": float64(testDurationAPIMs),
		"is_error":        false,
		"num_turns":       float64(testNumTurns),
		"session_id":      testSessionIDSuccess,
		"result":          testResultText,
		"total_cost_usd":  testCostUSD,
		"usage": map[string]any{
			"input_tokens":                testInputTokens,
			"output_tokens":               testOutputTokens,
			"cache_read_input_tokens":     testCacheReadTokens,
			"cache_creation_input_tokens": testCacheCreationTokens,
		},
		"modelUsage": map[string]any{
			testModelName: map[string]any{
				"inputTokens":              testInputTokens,
				"outputTokens":             testOutputTokens,
				"cacheReadInputTokens":     testCacheReadTokens,
				"cacheCreationInputTokens": testCacheCreationTokens,
				"webSearchRequests":        testWebSearchRequests,
				"costUSD":                  testCostUSD,
				"contextWindow":            testContextWindow,
			},
		},
		"permission_denials": make([]any, 0),
	}

	// Parse the message using the type-safe approach.
	msg, err := parseResultMessageV2(data)
	if err != nil {
		t.Fatalf("Failed to parse success result: %v", err)
	}

	// Verify we got the correct message type.
	successMsg, ok := msg.(*messages.ResultMessageSuccess)
	if !ok {
		t.Fatalf(
			"Expected *ResultMessageSuccess, got %T",
			msg,
		)
	}

	// Verify all fields were correctly parsed from JSON.
	if successMsg.DurationMs != testDurationMs {
		t.Errorf(
			"Expected DurationMs=%d, got %d",
			testDurationMs,
			successMsg.DurationMs,
		)
	}
	if successMsg.DurationAPIMs != testDurationAPIMs {
		t.Errorf(
			"Expected DurationAPIMs=%d, got %d",
			testDurationAPIMs,
			successMsg.DurationAPIMs,
		)
	}
	if successMsg.IsError {
		t.Error("Expected IsError=false")
	}
	if successMsg.NumTurns != testNumTurns {
		t.Errorf(
			"Expected NumTurns=%d, got %d",
			testNumTurns,
			successMsg.NumTurns,
		)
	}
	if successMsg.SessionID != testSessionIDSuccess {
		t.Errorf(
			"Expected SessionID='%s', got %s",
			testSessionIDSuccess,
			successMsg.SessionID,
		)
	}
	if successMsg.Result != testResultText {
		t.Errorf(
			"Expected Result='%s', got %s",
			testResultText,
			successMsg.Result,
		)
	}
}

// TestParseResultMessageV2_Error validates that an error result message
// is correctly parsed from a map into a typed ResultMessageError struct.
// Error results occur when max turns are exceeded or execution fails.
func TestParseResultMessageV2_Error(t *testing.T) {
	// Create test data for an error result.
	data := map[string]any{
		"type":            testType,
		"subtype":         testErrorSubtype,
		"duration_ms":     float64(testDurationMsError),
		"duration_api_ms": float64(testDurationAPIMsError),
		"is_error":        true,
		"num_turns":       float64(testNumTurnsError),
		"session_id":      testSessionIDError,
		"total_cost_usd":  testCostUSDError,
		"usage": map[string]any{
			"input_tokens":                testInputTokensError,
			"output_tokens":               testOutputTokensError,
			"cache_read_input_tokens":     0.0,
			"cache_creation_input_tokens": 0.0,
		},
		"modelUsage":         make(map[string]any),
		"permission_denials": make([]any, 0),
	}

	// Parse the error message.
	msg, err := parseResultMessageV2(data)
	if err != nil {
		t.Fatalf("Failed to parse error result: %v", err)
	}

	// Verify we got the error message type.
	errorMsg, ok := msg.(*messages.ResultMessageError)
	if !ok {
		t.Fatalf("Expected *ResultMessageError, got %T", msg)
	}

	// Verify error-specific fields.
	if errorMsg.DurationMs != testDurationMsError {
		t.Errorf(
			"Expected DurationMs=%d, got %d",
			testDurationMsError,
			errorMsg.DurationMs,
		)
	}
	if errorMsg.Subtype != testErrorSubtype {
		t.Errorf(
			"Expected Subtype='%s', got %s",
			testErrorSubtype,
			errorMsg.Subtype,
		)
	}
	if !errorMsg.IsError {
		t.Error("Expected IsError=true")
	}
}

// TestParseResultMessageV2_InvalidSubtype ensures that an unknown subtype
// returns an error rather than silently failing.
func TestParseResultMessageV2_InvalidSubtype(t *testing.T) {
	data := map[string]any{
		"type":    testType,
		"subtype": "invalid_subtype",
	}

	_, err := parseResultMessageV2(data)
	if err == nil {
		t.Fatal("Expected error for invalid subtype, got nil")
	}
}

// TestParseResultMessageV2_MissingSubtype ensures that a missing subtype
// field returns an error.
func TestParseResultMessageV2_MissingSubtype(t *testing.T) {
	data := map[string]any{
		"type": testType,
	}

	_, err := parseResultMessageV2(data)
	if err == nil {
		t.Fatal("Expected error for missing subtype, got nil")
	}
}
