package parse

import (
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser
type Adapter struct{}

// Verify interface compliance at compile time
var _ ports.MessageParser = (*Adapter)(nil)

// NewAdapter creates a new message parser adapter
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse implements ports.MessageParser
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
	msg := &messages.UserMessage{}

	// Parse content
	if content, ok := data["content"].(string); ok {
		msg.Content = messages.StringContent(content)
	} else if contentBlocks, ok := data["content"].([]any); ok {
		blocks := make([]messages.ContentBlock, 0, len(contentBlocks))
		for _, item := range contentBlocks {
			block, err := a.parseContentBlock(item)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
		msg.Content = messages.BlockListContent(blocks)
	}

	// Parse parent_tool_use_id
	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func (a *Adapter) parseSystemMessage(data map[string]any) (messages.Message, error) {
	msg := &messages.SystemMessage{}

	if subtype, ok := data["subtype"].(string); ok {
		msg.Subtype = subtype
	}

	if msgData, ok := data["data"].(map[string]any); ok {
		msg.Data = msgData
	}

	return msg, nil
}

func (a *Adapter) parseResultMessage(data map[string]any) (messages.Message, error) {
	msg := &messages.ResultMessage{}

	if subtype, ok := data["subtype"].(string); ok {
		msg.Subtype = subtype
	}

	if duration, ok := data["duration_ms"].(float64); ok {
		msg.DurationMs = int(duration)
	}

	if durationAPI, ok := data["duration_api_ms"].(float64); ok {
		msg.DurationAPIMs = int(durationAPI)
	}

	if isError, ok := data["is_error"].(bool); ok {
		msg.IsError = isError
	}

	if numTurns, ok := data["num_turns"].(float64); ok {
		msg.NumTurns = int(numTurns)
	}

	if sessionID, ok := data["session_id"].(string); ok {
		msg.SessionID = sessionID
	}

	if cost, ok := data["total_cost_usd"].(float64); ok {
		msg.TotalCostUSD = &cost
	}

	if usage, ok := data["usage"].(map[string]any); ok {
		msg.Usage = usage
	}

	if result, ok := data["result"].(string); ok {
		msg.Result = &result
	}

	return msg, nil
}

func (a *Adapter) parseStreamEvent(data map[string]any) (messages.Message, error) {
	msg := &messages.StreamEvent{}

	if uuid, ok := data["uuid"].(string); ok {
		msg.UUID = uuid
	}

	if sessionID, ok := data["session_id"].(string); ok {
		msg.SessionID = sessionID
	}

	if event, ok := data["event"].(map[string]any); ok {
		msg.Event = event
	}

	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func (a *Adapter) parseAssistantMessage(data map[string]any) (messages.Message, error) {
	msg := &messages.AssistantMessage{}

	// Parse message content
	if msgData, ok := data["message"].(map[string]any); ok {
		// Parse content blocks
		if contentArray, ok := msgData["content"].([]any); ok {
			blocks := make([]messages.ContentBlock, 0, len(contentArray))
			for _, item := range contentArray {
				block, err := a.parseContentBlock(item)
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, block)
			}
			msg.Content = blocks
		}

		// Parse model
		if model, ok := msgData["model"].(string); ok {
			msg.Model = model
		}
	}

	// Parse parent_tool_use_id
	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func (a *Adapter) parseContentBlock(item any) (messages.ContentBlock, error) {
	block, ok := item.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid content block")
	}

	blockType, _ := block["type"].(string)
	switch blockType {
	case "text":
		text, _ := block["text"].(string)

		return messages.TextBlock{Text: text}, nil

	case "thinking":
		thinking, _ := block["thinking"].(string)
		signature, _ := block["signature"].(string)

		return messages.ThinkingBlock{
			Thinking:  thinking,
			Signature: signature,
		}, nil

	case "tool_use":
		id, _ := block["id"].(string)
		name, _ := block["name"].(string)
		input, _ := block["input"].(map[string]any)

		return messages.ToolUseBlock{
			ID:    id,
			Name:  name,
			Input: input,
		}, nil

	case "tool_result":
		toolUseID, _ := block["tool_use_id"].(string)
		content := block["content"]
		var isError *bool
		if err, ok := block["is_error"].(bool); ok {
			isError = &err
		}

		return messages.ToolResultBlock{
			ToolUseID: toolUseID,
			Content:   content,
			IsError:   isError,
		}, nil

	default:
		return nil, fmt.Errorf("unknown content block type: %s", blockType)
	}
}
