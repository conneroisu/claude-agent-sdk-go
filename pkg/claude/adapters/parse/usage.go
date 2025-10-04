package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// parseUsageStats parses usage statistics from raw data.
func parseUsageStats(data any) (messages.UsageStats, error) {
	if data == nil {
		return messages.UsageStats{}, nil
	}

	usageMap, ok := data.(map[string]any)
	if !ok {
		return messages.UsageStats{}, fmt.Errorf("usage must be an object")
	}

	inputTokens, _ := usageMap["input_tokens"].(float64)
	outputTokens, _ := usageMap["output_tokens"].(float64)
	cacheReadInputTokens, _ := usageMap["cache_read_input_tokens"].(float64)
	cacheCreationInputTokens, _ := usageMap["cache_creation_input_tokens"].(float64)

	return messages.UsageStats{
		InputTokens:              int(inputTokens),
		OutputTokens:             int(outputTokens),
		CacheReadInputTokens:     int(cacheReadInputTokens),
		CacheCreationInputTokens: int(cacheCreationInputTokens),
	}, nil
}

// parseModelUsage parses per-model usage statistics.
func parseModelUsage(data any) (map[string]messages.ModelUsage, error) {
	if data == nil {
		return make(map[string]messages.ModelUsage), nil
	}

	modelUsageMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("modelUsage must be an object")
	}

	result := make(map[string]messages.ModelUsage)
	for modelName, usageData := range modelUsageMap {
		usageMap, ok := usageData.(map[string]any)
		if !ok {
			continue
		}

		result[modelName] = parseModelUsageEntry(usageMap)
	}

	return result, nil
}

func parseModelUsageEntry(usageMap map[string]any) messages.ModelUsage {
	inputTokens, _ := usageMap["inputTokens"].(float64)
	outputTokens, _ := usageMap["outputTokens"].(float64)
	cacheReadInputTokens, _ := usageMap["cacheReadInputTokens"].(float64)
	cacheCreationInputTokens, _ := usageMap["cacheCreationInputTokens"].(float64)
	webSearchRequests, _ := usageMap["webSearchRequests"].(float64)
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

// parsePermissionDenials parses array of permission denials.
func parsePermissionDenials(data any) ([]messages.PermissionDenial, error) {
	if data == nil {
		return []messages.PermissionDenial{}, nil
	}

	denialsArray, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("permission_denials must be an array")
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
