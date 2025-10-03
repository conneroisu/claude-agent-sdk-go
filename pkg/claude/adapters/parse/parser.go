package parse

import (
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser
// This is an INFRASTRUCTURE adapter - handles low-level message parsing.
type Adapter struct{}

// Verify interface compliance at compile time.
var _ ports.MessageParser = (*Adapter)(nil)

func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse implements ports.MessageParser.
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing type field")
	}
	switch msgType {
	case "user":
		return a.parseUserMessage(data)
	case "assistant":
		return a.parseAssistantMessage(data)
	case "system":
		return a.parseSystemMessage(data)
	case "result":
		return a.parseResultMessage(data)
	case "stream_event":
		return a.parseStreamEvent(data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}
func (a *Adapter) parseUserMessage(data map[string]any) (messages.Message, error) {
	msg, _ := data["message"].(map[string]any)

	// Parse content (can be string or array of blocks)
	var content messages.MessageContent
	if contentStr, ok := msg["content"].(string); ok {
		content = messages.StringContent(contentStr)
	} else if contentArr, ok := msg["content"].([]any); ok {
		blocks, err := parseContentBlocks(contentArr)
		if err != nil {
			return nil, fmt.Errorf("parse user message content blocks: %w", err)
		}
		content = messages.BlockListContent(blocks)
	} else {
		return nil, fmt.Errorf("user message content must be string or array")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.UserMessage{
		Content:         content,
		ParentToolUseID: parentToolUseID,
	}, nil
}

func (a *Adapter) parseSystemMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("system message missing subtype field")
	}

	// Data field is intentionally kept as map[string]any
	// Users can parse it into specific SystemMessageData types if needed
	// (SystemMessageInit, SystemMessageCompactBoundary)
	systemData, _ := data["data"].(map[string]any)
	if systemData == nil {
		systemData = make(map[string]any)
	}

	return &messages.SystemMessage{
		Subtype: subtype,
		Data:    systemData,
	}, nil
}

func (a *Adapter) parseResultMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("result message missing subtype field")
	}

	// Type-safe approach: marshal map to JSON, then unmarshal into typed struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result message: %w", err)
	}

	switch subtype {
	case "success":
		var result messages.ResultMessageSuccess
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal success result: %w", err)
		}

		return &result, nil

	case "error_max_turns", "error_during_execution":
		var result messages.ResultMessageError
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal error result: %w", err)
		}

		return &result, nil

	default:
		return nil, fmt.Errorf("unknown result subtype: %s", subtype)
	}
}

func (a *Adapter) parseStreamEvent(data map[string]any) (messages.Message, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing uuid field")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing session_id field")
	}

	event, ok := data["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("stream event missing event field")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.StreamEvent{
		UUID:            uuid,
		SessionID:       sessionID,
		Event:           event, // Keep as map[string]any (raw Anthropic API event)
		ParentToolUseID: parentToolUseID,
	}, nil
}
func (a *Adapter) parseAssistantMessage(data map[string]any) (messages.Message, error) {
	// Parse content blocks
	msg, _ := data["message"].(map[string]any)
	contentArray, _ := msg["content"].([]any)
	var blocks []messages.ContentBlock
	for _, item := range contentArray {
		block, _ := item.(map[string]any)
		blockType, _ := block["type"].(string)
		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			blocks = append(blocks, messages.TextBlock{
				Type: "text",
				Text: text,
			})
		case "thinking":
			thinking, _ := block["thinking"].(string)
			signature, _ := block["signature"].(string)
			blocks = append(blocks, messages.ThinkingBlock{
				Type:      "thinking",
				Thinking:  thinking,
				Signature: signature,
			})
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			blocks = append(blocks, messages.ToolUseBlock{
				Type:  "tool_use",
				ID:    id,
				Name:  name,
				Input: input,
			})
		case "tool_result":
			toolResultBlock, err := parseToolResultBlock(block)
			if err != nil {
				// Skip invalid tool result blocks
				continue
			}
			blocks = append(blocks, toolResultBlock)
		}
	}
	model, _ := msg["model"].(string)
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")
	return &messages.AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}

// Helper function for extracting optional string pointers
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}
	return nil
}

// parseUsageStats parses usage statistics from raw data
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

// parseModelUsage parses per-model usage statistics
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

		inputTokens, _ := usageMap["inputTokens"].(float64)
		outputTokens, _ := usageMap["outputTokens"].(float64)
		cacheReadInputTokens, _ := usageMap["cacheReadInputTokens"].(float64)
		cacheCreationInputTokens, _ := usageMap["cacheCreationInputTokens"].(float64)
		webSearchRequests, _ := usageMap["webSearchRequests"].(float64)
		costUSD, _ := usageMap["costUSD"].(float64)
		contextWindow, _ := usageMap["contextWindow"].(float64)

		result[modelName] = messages.ModelUsage{
			InputTokens:              int(inputTokens),
			OutputTokens:             int(outputTokens),
			CacheReadInputTokens:     int(cacheReadInputTokens),
			CacheCreationInputTokens: int(cacheCreationInputTokens),
			WebSearchRequests:        int(webSearchRequests),
			CostUSD:                  costUSD,
			ContextWindow:            int(contextWindow),
		}
	}

	return result, nil
}

// parsePermissionDenials parses array of permission denials
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

// parseContentBlocks parses an array of content blocks
func parseContentBlocks(contentArr []any) ([]messages.ContentBlock, error) {
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		block, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content block must be an object")
		}

		blockType, ok := block["type"].(string)
		if !ok {
			return nil, fmt.Errorf("content block missing type field")
		}

		switch blockType {
		case "text":
			textBlock, err := parseTextBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, textBlock)

		case "thinking":
			thinkingBlock, err := parseThinkingBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, thinkingBlock)

		case "tool_use":
			toolUseBlock, err := parseToolUseBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolUseBlock)

		case "tool_result":
			toolResultBlock, err := parseToolResultBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolResultBlock)

		default:
			return nil, fmt.Errorf("unknown content block type: %s", blockType)
		}
	}

	return blocks, nil
}

// parseTextBlock parses a text content block
func parseTextBlock(block map[string]any) (messages.TextBlock, error) {
	text, ok := block["text"].(string)
	if !ok {
		return messages.TextBlock{}, fmt.Errorf("text block missing text field")
	}

	return messages.TextBlock{
		Type: "text",
		Text: text,
	}, nil
}

// parseThinkingBlock parses a thinking content block
func parseThinkingBlock(block map[string]any) (messages.ThinkingBlock, error) {
	thinking, ok := block["thinking"].(string)
	if !ok {
		return messages.ThinkingBlock{}, fmt.Errorf("thinking block missing thinking field")
	}

	signature, _ := block["signature"].(string)

	return messages.ThinkingBlock{
		Type:      "thinking",
		Thinking:  thinking,
		Signature: signature,
	}, nil
}

// parseToolUseBlock parses a tool_use content block
func parseToolUseBlock(block map[string]any) (messages.ToolUseBlock, error) {
	id, ok := block["id"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing id field")
	}

	name, ok := block["name"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing name field")
	}

	input, ok := block["input"].(map[string]any)
	if !ok {
		// Input can be missing or null
		input = make(map[string]any)
	}

	return messages.ToolUseBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResultBlock parses a tool_result content block
func parseToolResultBlock(block map[string]any) (messages.ToolResultBlock, error) {
	toolUseID, ok := block["tool_use_id"].(string)
	if !ok {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result block missing tool_use_id field")
	}

	// Parse content (can be string or array of content blocks)
	var content messages.ToolResultContent
	if contentStr, ok := block["content"].(string); ok {
		content = messages.ToolResultStringContent(contentStr)
	} else if contentArr, ok := block["content"].([]any); ok {
		// Tool result content can be an array of raw content blocks (maps)
		blockMaps := make([]map[string]any, 0, len(contentArr))
		for _, item := range contentArr {
			if blockMap, ok := item.(map[string]any); ok {
				blockMaps = append(blockMaps, blockMap)
			}
		}
		content = messages.ToolResultBlockListContent(blockMaps)
	} else {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result content must be string or array")
	}

	// is_error is optional
	var isError *bool
	if isErrorVal, ok := block["is_error"].(bool); ok {
		isError = &isErrorVal
	}

	return messages.ToolResultBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}
