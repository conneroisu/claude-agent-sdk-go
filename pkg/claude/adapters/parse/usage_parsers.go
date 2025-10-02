// Package parse provides message parsing adapters for the Claude SDK.
package parse

import (
	"errors"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUsageStats parses usage statistics from raw data.
// Token counts are essential for tracking API costs and context
// window usage. Cache tokens help optimize costs by reusing
// previously processed content.
func parseUsageStats(data any) (messages.UsageStats, error) {
	if data == nil {
		return messages.UsageStats{}, nil
	}

	usageMap, ok := data.(map[string]any)
	if !ok {
		return messages.UsageStats{}, errors.New("usage must be an object")
	}

	// Extract token counts - JSON numbers are float64 by default
	inputTokens, _ := usageMap["input_tokens"].(float64)
	outputTokens, _ := usageMap["output_tokens"].(float64)
	cacheReadInputTokens, _ :=
		usageMap["cache_read_input_tokens"].(float64)
	cacheCreationInputTokens, _ :=
		usageMap["cache_creation_input_tokens"].(float64)

	return messages.UsageStats{
		InputTokens:              int(inputTokens),
		OutputTokens:             int(outputTokens),
		CacheReadInputTokens:     int(cacheReadInputTokens),
		CacheCreationInputTokens: int(cacheCreationInputTokens),
	}, nil
}

// parseModelUsage parses per-model usage statistics.
// Multi-model sessions track usage separately for each model to
// provide accurate cost attribution and performance insights.
func parseModelUsage(
	data any,
) (map[string]messages.ModelUsage, error) {
	if data == nil {
		return make(map[string]messages.ModelUsage), nil
	}

	modelUsageMap, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("modelUsage must be an object")
	}

	result := make(map[string]messages.ModelUsage)
	for modelName, usageData := range modelUsageMap {
		usageMap, ok := usageData.(map[string]any)
		if !ok {
			continue
		}

		modelUsage := parseModelUsageEntry(usageMap)
		result[modelName] = modelUsage
	}

	return result, nil
}

// parseModelUsageEntry parses a single model usage entry
func parseModelUsageEntry(
	usageMap map[string]any,
) messages.ModelUsage {
	// Extract all usage metrics for this specific model
	inputTokens, _ := usageMap["inputTokens"].(float64)
	outputTokens, _ := usageMap["outputTokens"].(float64)
	cacheReadInputTokens, _ :=
		usageMap["cacheReadInputTokens"].(float64)
	cacheCreationInputTokens, _ :=
		usageMap["cacheCreationInputTokens"].(float64)
	webSearchRequests, _ :=
		usageMap["webSearchRequests"].(float64)
	costUSD, _ := usageMap["costUSD"].(float64)
	contextWindow, _ := usageMap["contextWindow"].(float64)

	return messages.ModelUsage{
		InputTokens:              int(inputTokens),
		OutputTokens:             int(outputTokens),
		CacheReadInputTokens:     int(cacheReadInputTokens),
		CacheCreationInputTokens: int(cacheCreationInputTokens),
		WebSearchRequests:        int(webSearchRequests),
		CostUSD:                  costUSD,
		ContextWindow:            int(contextWindow),
	}
}

// parsePermissionDenials parses array of permission denials
func parsePermissionDenials(
	data any,
) ([]messages.PermissionDenial, error) {
	if data == nil {
		return make([]messages.PermissionDenial, 0), nil
	}

	denialsArray, ok := data.([]any)
	if !ok {
		return nil, errors.New("permission_denials must be an array")
	}

	result := make([]messages.PermissionDenial, 0, len(denialsArray))
	for _, denialData := range denialsArray {
		denialMap, ok := denialData.(map[string]any)
		if !ok {
			continue
		}

		toolName, _ := denialMap["tool_name"].(string)
		toolUseID, _ := denialMap["tool_use_id"].(string)
		toolInput, _ := denialMap["tool_input"].(map[string]any)

		result = append(result, messages.PermissionDenial{
			ToolName:  toolName,
			ToolUseID: toolUseID,
			ToolInput: toolInput,
		})
	}

	return result, nil
}
