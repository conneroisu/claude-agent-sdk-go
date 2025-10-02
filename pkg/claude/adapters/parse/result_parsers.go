// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// resultMessageFields holds common fields for result messages
type resultMessageFields struct {
	subtype           string
	durationMs        int
	durationAPIMs     int
	isError           bool
	numTurns          int
	sessionID         string
	totalCostUSD      float64
	usage             messages.UsageStats
	modelUsage        map[string]messages.ModelUsage
	permissionDenials []messages.PermissionDenial
}

// parseResultMessageFields extracts common fields from result
// message data
func parseResultMessageFields(
	data map[string]any,
) (*resultMessageFields, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, errors.New("result message missing subtype field")
	}

	// Parse common fields shared across all result subtypes
	durationMs, _ := data["duration_ms"].(float64)
	durationAPIMs, _ := data["duration_api_ms"].(float64)
	isError, _ := data["is_error"].(bool)
	numTurns, _ := data["num_turns"].(float64)
	sessionID, _ := data["session_id"].(string)
	totalCostUSD, _ := data["total_cost_usd"].(float64)

	usage, err := parseUsageStats(data["usage"])
	if err != nil {
		return nil, fmt.Errorf("parse usage stats: %w", err)
	}

	modelUsage, err := parseModelUsage(data["modelUsage"])
	if err != nil {
		return nil, fmt.Errorf("parse model usage: %w", err)
	}

	permissionDenials, err := parsePermissionDenials(
		data["permission_denials"],
	)
	if err != nil {
		return nil, fmt.Errorf("parse permission denials: %w", err)
	}

	return &resultMessageFields{
		subtype:           subtype,
		durationMs:        int(durationMs),
		durationAPIMs:     int(durationAPIMs),
		isError:           isError,
		numTurns:          int(numTurns),
		sessionID:         sessionID,
		totalCostUSD:      totalCostUSD,
		usage:             usage,
		modelUsage:        modelUsage,
		permissionDenials: permissionDenials,
	}, nil
}

// buildResultMessage constructs the appropriate result message type
// based on subtype
func buildResultMessage(
	fields *resultMessageFields,
	data map[string]any,
) (messages.Message, error) {
	switch fields.subtype {
	case "success":
		return buildSuccessMessage(fields, data)
	case "error_max_turns", "error_during_execution":
		return buildErrorMessage(fields)
	default:
		return nil, fmt.Errorf(
			"unknown result subtype: %s",
			fields.subtype,
		)
	}
}

// buildSuccessMessage builds a success result message
func buildSuccessMessage(
	fields *resultMessageFields,
	data map[string]any,
) (messages.Message, error) {
	result, _ := data["result"].(string)

	return &messages.ResultMessageSuccess{
		Subtype:           fields.subtype,
		DurationMs:        fields.durationMs,
		DurationAPIMs:     fields.durationAPIMs,
		IsError:           fields.isError,
		NumTurns:          fields.numTurns,
		SessionID:         fields.sessionID,
		Result:            result,
		TotalCostUSD:      fields.totalCostUSD,
		Usage:             fields.usage,
		ModelUsage:        fields.modelUsage,
		PermissionDenials: fields.permissionDenials,
	}, nil
}

// buildErrorMessage builds an error result message
func buildErrorMessage(
	fields *resultMessageFields,
) (messages.Message, error) {
	return &messages.ResultMessageError{
		Subtype:           fields.subtype,
		DurationMs:        fields.durationMs,
		DurationAPIMs:     fields.durationAPIMs,
		IsError:           fields.isError,
		NumTurns:          fields.numTurns,
		SessionID:         fields.sessionID,
		TotalCostUSD:      fields.totalCostUSD,
		Usage:             fields.usage,
		ModelUsage:        fields.modelUsage,
		PermissionDenials: fields.permissionDenials,
	}, nil
}
