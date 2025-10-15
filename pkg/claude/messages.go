package claude

import (
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

const (
	// MessageTypeToolResult is the message type for tool result content blocks.
	MessageTypeToolResult = "tool_result"
	// ContentBlockStart is the content block start event type.
	ContentBlockStart = "content_block_start"
	// ContentBlockDelta is the content block delta event type.
	ContentBlockDelta = "content_block_delta"
	// ControlRequest is the control request message type.
	ControlRequest = "control_request"
)

// SDKMessage is the interface all SDK messages implement.
type SDKMessage interface {
	// UUID returns the unique identifier for this message.
	UUID() UUID
	// SessionID returns the session identifier for this message.
	SessionID() string
	// Type returns the message type string.
	Type() string
	sdkMessage()
}

// BaseMessage contains common message fields.
type BaseMessage struct {
	UUIDField      UUID   `json:"uuid"`
	SessionIDField string `json:"session_id"`
}

func (b BaseMessage) UUID() UUID        { return b.UUIDField }
func (b BaseMessage) SessionID() string { return b.SessionIDField }
func (BaseMessage) sdkMessage()         {}

// SDKUserMessage represents a user message.
type SDKUserMessage struct {
	BaseMessage
	TypeField       string         `json:"type"`
	Message         APIUserMessage `json:"message"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	IsSynthetic     bool           `json:"isSynthetic,omitempty"`
}

func (m SDKUserMessage) Type() string {
	if m.TypeField != "" {
		return m.TypeField
	}

	return "user"
}

// APIUserMessage represents the actual user message content.
type APIUserMessage struct {
	Role    string         `json:"role"` // "user"
	Content []ContentBlock `json:"content"`
}

// UnmarshalJSON custom unmarshaler for APIUserMessage.
func (m *APIUserMessage) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse APIUserMessage JSON",
			err,
		).WithMessageType("user")
	}

	m.Role = aux.Role

	// Decode content blocks
	blocks, err := decodeContentBlocks(aux.Content)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"failed to decode user message content blocks",
			err,
		).WithMessageType("user")
	}
	m.Content = blocks

	return nil
}

// ContentBlock represents different content types.
type ContentBlock interface {
	contentBlock()
}

// TextContentBlock represents text content.
type TextContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

func (TextContentBlock) contentBlock() {}

// ImageContentBlock represents image content.
type ImageContentBlock struct {
	Type   string      `json:"type"` // "image"
	Source ImageSource `json:"source"`
}

func (ImageContentBlock) contentBlock() {}

type ImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ToolUseContentBlock represents tool use.
type ToolUseContentBlock struct {
	Type  string    `json:"type"` // "tool_use"
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Input JSONValue `json:"input"`
}

func (ToolUseContentBlock) contentBlock() {}

// ToolResultContent represents either a string or nested content blocks.
type ToolResultContent struct {
	Text   *string        // Mutually exclusive with Blocks
	Blocks []ContentBlock // Present when content is structured
}

// MarshalJSON encodes the union as either a string or []ContentBlock.
func (c ToolResultContent) MarshalJSON() ([]byte, error) {
	switch {
	case c.Text != nil && len(c.Blocks) == 0:
		return json.Marshal(c.Text)
	case c.Text == nil && len(c.Blocks) > 0:
		return json.Marshal(c.Blocks)
	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"tool result content must be either text or blocks, "+
				"not both or neither",
			nil,
		).WithMessageType(MessageTypeToolResult)
	}
}

// UnmarshalJSON decodes the union form.
func (c *ToolResultContent) UnmarshalJSON(data []byte) error {
	// Attempt string decode first
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		c.Text = &text
		c.Blocks = nil

		return nil
	}

	if blocks, err := decodeContentBlocks(data); err == nil {
		c.Blocks = blocks
		c.Text = nil

		return nil
	}

	return clauderrs.NewProtocolError(
		clauderrs.ErrCodeInvalidMessage,
		"tool result content must be string or []ContentBlock",
		nil,
	).WithMessageType(MessageTypeToolResult)
}

// ToolResultContentBlock represents tool result with strongly typed content
// union.
type ToolResultContentBlock struct {
	Type      string             `json:"type"` // "tool_result"
	ToolUseID string             `json:"tool_use_id"`
	Content   *ToolResultContent `json:"content,omitempty"`
	IsError   bool               `json:"is_error,omitempty"`
}

func (ToolResultContentBlock) contentBlock() {}

// ThinkingBlock represents extended thinking content from Claude.
type ThinkingBlock struct {
	Type     string `json:"type"` // "thinking"
	Thinking string `json:"thinking"`
}

func (ThinkingBlock) contentBlock() {}

// TextBlock represents text content.
type TextBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

// decodeContentBlocks converts raw JSON array into typed content blocks.
func decodeContentBlocks(data []byte) ([]ContentBlock, error) {
	var rawBlocks []json.RawMessage
	if err := json.Unmarshal(data, &rawBlocks); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse content blocks array",
			err,
		)
	}

	blocks := make([]ContentBlock, len(rawBlocks))
	for i, raw := range rawBlocks {
		block, err := decodeContentBlock(raw)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeInvalidMessage,
				fmt.Sprintf("failed to decode content block at index %d", i),
				err,
			)
		}
		blocks[i] = block
	}

	return blocks, nil
}

func decodeContentBlock(data []byte) (ContentBlock, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse content block type envelope",
			err,
		)
	}

	switch envelope.Type {
	case "text":
		var block TextContentBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse text content block",
				err,
			).WithMessageType("text")
		}

		return block, nil
	case "thinking":
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse thinking block",
				err,
			).WithMessageType("thinking")
		}

		return block, nil
	case "image":
		var block ImageContentBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse image content block",
				err,
			).WithMessageType("image")
		}

		return block, nil
	case "tool_use":
		var block ToolUseContentBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse tool_use content block",
				err,
			).WithMessageType("tool_use")
		}

		return block, nil
	case MessageTypeToolResult:
		var block ToolResultContentBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse tool_result content block",
				err,
			).WithMessageType(MessageTypeToolResult)
		}

		return block, nil
	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf("unsupported content block type: %s", envelope.Type),
			nil,
		).WithMessageType(envelope.Type)
	}
}

// SDKAssistantMessage represents an assistant response.
type SDKAssistantMessage struct {
	BaseMessage
	Message         APIAssistantMessage `json:"message"`
	ParentToolUseID *string             `json:"parent_tool_use_id,omitempty"`
}

func (SDKAssistantMessage) Type() string { return "assistant" }

// APIAssistantMessage represents the actual assistant message.
type APIAssistantMessage struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason,omitempty"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// UnmarshalJSON custom unmarshaler for APIAssistantMessage.
func (m *APIAssistantMessage) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID           string          `json:"id"`
		Type         string          `json:"type"`
		Role         string          `json:"role"`
		Content      json.RawMessage `json:"content"`
		Model        string          `json:"model"`
		StopReason   *string         `json:"stop_reason,omitempty"`
		StopSequence *string         `json:"stop_sequence,omitempty"`
		Usage        Usage           `json:"usage"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse APIAssistantMessage JSON",
			err,
		).WithMessageType("assistant")
	}

	m.ID = aux.ID
	m.Type = aux.Type
	m.Role = aux.Role
	m.Model = aux.Model
	m.StopReason = aux.StopReason
	m.StopSequence = aux.StopSequence
	m.Usage = aux.Usage

	// Decode content blocks
	blocks, err := decodeContentBlocks(aux.Content)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"failed to decode assistant message content blocks",
			err,
		).WithMessageType("assistant").WithMessageID(aux.ID)
	}
	m.Content = blocks

	return nil
}

// SDKStreamEvent represents partial message streaming.
type SDKStreamEvent struct {
	BaseMessage
	Event           RawMessageStreamEvent `json:"-"`
	ParentToolUseID *string               `json:"parent_tool_use_id,omitempty"`
}

func (SDKStreamEvent) Type() string { return "stream_event" }

// UnmarshalJSON decodes the event union into a typed value.
func (e *SDKStreamEvent) UnmarshalJSON(data []byte) error {
	type Alias struct {
		BaseMessage
		Event           json.RawMessage `json:"event"`
		ParentToolUseID *string         `json:"parent_tool_use_id,omitempty"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse SDKStreamEvent JSON",
			err,
		).WithMessageType("stream_event").WithSessionID(aux.SessionIDField)
	}

	evt, err := decodeRawMessageStreamEvent(aux.Event)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"failed to decode stream event",
			err,
		).WithMessageType("stream_event").WithSessionID(aux.SessionIDField)
	}

	e.BaseMessage = aux.BaseMessage
	e.Event = evt
	e.ParentToolUseID = aux.ParentToolUseID

	return nil
}

// RawMessageStreamEvent captures the discriminated union of stream events.
type RawMessageStreamEvent interface {
	// EventType returns the type of stream event.
	EventType() string
}

type MessageStartEvent struct {
	Type    string              `json:"type"` // "message_start"
	Message APIAssistantMessage `json:"message"`
}

func (MessageStartEvent) EventType() string { return "message_start" }

// ContentBlockStartEvent represents a content block start event.
type ContentBlockStartEvent struct {
	Type         string       `json:"type"` // "content_block_start"
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

// EventType returns the type of event.
func (ContentBlockStartEvent) EventType() string {
	return ContentBlockStart
}

type ContentBlockDeltaEvent struct {
	Type  string       `json:"type"` // "content_block_delta"
	Index int          `json:"index"`
	Delta ContentDelta `json:"delta"`
}

func (ContentBlockDeltaEvent) EventType() string {
	return ContentBlockDelta
}

type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

func (ContentBlockStopEvent) EventType() string { return "content_block_stop" }

type MessageDeltaEvent struct {
	Type  string `json:"type"` // "message_delta"
	Usage *Usage `json:"usage,omitempty"`
}

func (MessageDeltaEvent) EventType() string { return "message_delta" }

type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

func (MessageStopEvent) EventType() string { return "message_stop" }

// ContentDelta represents partial updates to a text or tool block.
type ContentDelta struct {
	TextDelta *string `json:"text_delta,omitempty"`
	// Additional delta fields handled as needed (e.g., tool call streaming)
}

// decodeContentDelta converts raw JSON into a typed delta representation.
func decodeContentDelta(data []byte) (ContentDelta, error) {
	var envelope struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ContentDelta{}, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse content delta envelope",
			err,
		)
	}

	switch envelope.Type {
	case "text_delta", "input_json_delta":
		// "input_json_delta" also carries text payloads for current protocol.
		if envelope.Text == "" {
			return ContentDelta{}, clauderrs.NewProtocolError(
				clauderrs.ErrCodeInvalidMessage,
				"text delta payload missing text field",
				nil,
			).WithMessageType(envelope.Type)
		}
		text := envelope.Text

		return ContentDelta{TextDelta: &text}, nil
	default:
		return ContentDelta{}, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf(
				"unsupported content delta type: %s",
				envelope.Type,
			),
			nil,
		).WithMessageType(envelope.Type)
	}
}

// decodeRawMessageStreamEvent converts a raw JSON event into the proper struct.
func decodeRawMessageStreamEvent(
	data json.RawMessage,
) (RawMessageStreamEvent, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(data, &envelope)
	if err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse stream event type envelope",
			err,
		)
	}

	switch envelope.Type {
	case "message_start":
		var evt MessageStartEvent
		err = json.Unmarshal(data, &evt)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse message_start event",
				err,
			).WithMessageType("message_start")
		}

		return evt, nil
	case "content_block_start":
		var raw struct {
			Type         string          `json:"type"`
			Index        int             `json:"index"`
			ContentBlock json.RawMessage `json:"content_block"`
		}
		err = json.Unmarshal(data, &raw)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse content_block_start event",
				err,
			).WithMessageType(ContentBlockStart)
		}
		var block ContentBlock
		block, err = decodeContentBlock(raw.ContentBlock)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeInvalidMessage,
				fmt.Sprintf(
					"failed to decode content block in content_block_start "+
						"at index %d",
					raw.Index,
				),
				err,
			).WithMessageType(ContentBlockStart)
		}

		return ContentBlockStartEvent{
			Type:         raw.Type,
			Index:        raw.Index,
			ContentBlock: block,
		}, nil
	case "content_block_delta":
		var raw struct {
			Type  string          `json:"type"`
			Index int             `json:"index"`
			Delta json.RawMessage `json:"delta"`
		}
		err = json.Unmarshal(data, &raw)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse content_block_delta event",
				err,
			).WithMessageType(ContentBlockDelta)
		}
		delta, err := decodeContentDelta(raw.Delta)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeInvalidMessage,
				fmt.Sprintf(
				"failed to decode delta in content_block_delta "+
					"at index %d",
				raw.Index,
			),
				err,
			).WithMessageType(ContentBlockDelta)
		}

		return ContentBlockDeltaEvent{
			Type:  raw.Type,
			Index: raw.Index,
			Delta: delta,
		}, nil
	case "content_block_stop":
		var evt ContentBlockStopEvent
		err = json.Unmarshal(data, &evt)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse content_block_stop event",
				err,
			).WithMessageType("content_block_stop")
		}

		return evt, nil
	case "message_delta":
		var evt MessageDeltaEvent
		err = json.Unmarshal(data, &evt)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse message_delta event",
				err,
			).WithMessageType("message_delta")
		}

		return evt, nil
	case "message_stop":
		var evt MessageStopEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse message_stop event",
				err,
			).WithMessageType("message_stop")
		}

		return evt, nil
	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf("unsupported stream event type: %s", envelope.Type),
			nil,
		).WithMessageType(envelope.Type)
	}
}

// SDKSystemMessage represents system information.
type SDKSystemMessage struct {
	BaseMessage
	Subtype string               `json:"subtype"`
	Data    map[string]JSONValue `json:"-"` // Custom data per subtype
}

func (SDKSystemMessage) Type() string { return "system" }

// SystemInitMessage represents initialization message.
type SystemInitMessage struct {
	SDKSystemMessage
	Agents         []string          `json:"agents"`
	APIKeySource   APIKeySource      `json:"apiKeySource"`
	Cwd            string            `json:"cwd"`
	Tools          []string          `json:"tools"`
	McpServers     []McpServerStatus `json:"mcp_servers"`
	Model          string            `json:"model"`
	PermissionMode PermissionMode    `json:"permissionMode"`
	SlashCommands  []string          `json:"slash_commands"`
	OutputStyle    string            `json:"output_style"`
}

// McpServerStatus represents MCP server status.
type McpServerStatus struct {
	Name       string         `json:"name"`
	Status string `json:"status"` // "connected", "failed", "needs-auth",
	// "pending"
	ServerInfo *McpServerInfo `json:"serverInfo,omitempty"`
}

type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SDKCompactBoundaryMessage represents compaction boundary.
type SDKCompactBoundaryMessage struct {
	SDKSystemMessage
	CompactMetadata CompactMetadata `json:"compact_metadata"`
}

type CompactMetadata struct {
	Trigger   string `json:"trigger"` // "manual" or "auto"
	PreTokens int    `json:"pre_tokens"`
}

// SDKResultMessage represents final query result.
type SDKResultMessage struct {
	BaseMessage
	Subtype           string                `json:"subtype"`
	DurationMS        int                   `json:"duration_ms"`
	DurationAPIMS     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             Usage                 `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []SDKPermissionDenial `json:"permission_denials"`
	Result *string `json:"result,omitempty"` // Only for success
}

func (SDKResultMessage) Type() string { return "result" }

const (
	ResultSubtypeSuccess              = "success"
	ResultSubtypeErrorMaxTurns        = "error_max_turns"
	ResultSubtypeErrorDuringExecution = "error_during_execution"
)

// SDKPermissionDenial represents a denied tool use.
type SDKPermissionDenial struct {
	ToolName  string               `json:"tool_name"`
	ToolUseID string               `json:"tool_use_id"`
	ToolInput map[string]JSONValue `json:"tool_input"`
}

// ============================================================================
// Control Protocol Messages
// ============================================================================

// SDKControlRequest represents control requests sent TO the Claude CLI.
// These are requests that the SDK sends to control CLI behavior.
type SDKControlRequest struct {
	BaseMessage
	RequestID string                `json:"request_id"`
	Request   ControlRequestVariant `json:"request"`
}

func (SDKControlRequest) Type() string { return ControlRequest }

// ControlRequestVariant is the interface for all control request variants.
type ControlRequestVariant interface {
	// Subtype returns the control request subtype string.
	Subtype() string
	controlRequestVariant()
}

// SDKControlInterruptRequest requests interruption of current execution.
type SDKControlInterruptRequest struct {
	SubtypeField string `json:"subtype"` // "interrupt"
}

func (r SDKControlInterruptRequest) Subtype() string {
	return "interrupt"
}
func (SDKControlInterruptRequest) controlRequestVariant() {}

// MarshalJSON ensures the subtype field is always set to "interrupt".
func (r SDKControlInterruptRequest) MarshalJSON() ([]byte, error) {
	type Alias SDKControlInterruptRequest
	return json.Marshal(&struct {
		SubtypeField string `json:"subtype"`
		*Alias
	}{
		SubtypeField: "interrupt",
		Alias:        (*Alias)(&r),
	})
}

// SDKControlInitializeRequest initializes the control session with hooks.
type SDKControlInitializeRequest struct {
	SubtypeField string               `json:"subtype"` // "initialize"
	Hooks        map[string]JSONValue `json:"hooks,omitempty"`
}

func (r SDKControlInitializeRequest) Subtype() string {
	return "initialize"
}
func (SDKControlInitializeRequest) controlRequestVariant() {}

// MarshalJSON ensures the subtype field is always set to "initialize".
func (r SDKControlInitializeRequest) MarshalJSON() ([]byte, error) {
	type Alias SDKControlInitializeRequest
	return json.Marshal(&struct {
		SubtypeField string `json:"subtype"`
		*Alias
	}{
		SubtypeField: "initialize",
		Alias:        (*Alias)(&r),
	})
}

// SDKControlSetPermissionModeRequest changes the permission mode.
type SDKControlSetPermissionModeRequest struct {
	SubtypeField string `json:"subtype"` // "set_permission_mode"
	Mode         string `json:"mode"`
}

// Subtype returns the Permission mode request subtype field.
func (SDKControlSetPermissionModeRequest) Subtype() string {
	return "set_permission_mode"
}
func (SDKControlSetPermissionModeRequest) controlRequestVariant() {}

// MarshalJSON ensures the subtype field is always set to "set_permission_mode".
func (r SDKControlSetPermissionModeRequest) MarshalJSON() ([]byte, error) {
	type Alias SDKControlSetPermissionModeRequest
	return json.Marshal(&struct {
		SubtypeField string `json:"subtype"`
		*Alias
	}{
		SubtypeField: "set_permission_mode",
		Alias:        (*Alias)(&r),
	})
}

// SDKControlMcpMessageRequest sends a message to an MCP server.
type SDKControlMcpMessageRequest struct {
	SubtypeField string    `json:"subtype"` // "mcp_message"
	ServerName   string    `json:"server_name"`
	Message      JSONValue `json:"message"`
}

func (SDKControlMcpMessageRequest) Subtype() string {
	return "mcp_message"
}
func (SDKControlMcpMessageRequest) controlRequestVariant() {}

// MarshalJSON ensures the subtype field is always set to "mcp_message".
func (r SDKControlMcpMessageRequest) MarshalJSON() ([]byte, error) {
	type Alias SDKControlMcpMessageRequest
	return json.Marshal(&struct {
		SubtypeField string `json:"subtype"`
		*Alias
	}{
		SubtypeField: "mcp_message",
		Alias:        (*Alias)(&r),
	})
}

// UnmarshalJSON custom unmarshaler for SDKControlRequest to handle
// the request variant.
func (r *SDKControlRequest) UnmarshalJSON(data []byte) error {
	type Alias struct {
		BaseMessage
		RequestID string          `json:"request_id"`
		Request   json.RawMessage `json:"request"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse SDKControlRequest JSON",
			err,
		).WithMessageType(ControlRequest)
	}

	r.BaseMessage = aux.BaseMessage
	r.RequestID = aux.RequestID

	// Decode the request variant
	variant, err := decodeControlRequestVariant(aux.Request)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"failed to decode control request variant",
			err,
		).WithMessageType(ControlRequest).WithRequestID(aux.RequestID)
	}
	r.Request = variant

	return nil
}

// decodeControlRequestVariant converts raw JSON into a typed control
// request variant.
func decodeControlRequestVariant(data []byte) (ControlRequestVariant, error) {
	var envelope struct {
		Subtype string `json:"subtype"`
	}
	err := json.Unmarshal(data, &envelope)
	if err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse control request subtype envelope",
			err,
		)
	}

	switch envelope.Subtype {
	case "interrupt":
		var req SDKControlInterruptRequest
		err = json.Unmarshal(data, &req)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse interrupt control request",
				err,
			).WithMessageType("interrupt")
		}

		return req, nil
	case "initialize":
		var req SDKControlInitializeRequest
		err = json.Unmarshal(data, &req)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse initialize control request",
				err,
			).WithMessageType("initialize")
		}

		return req, nil
	case "set_permission_mode":
		var req SDKControlSetPermissionModeRequest
		err = json.Unmarshal(data, &req)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse set_permission_mode control request",
				err,
			).WithMessageType("set_permission_mode")
		}

		return req, nil
	case "mcp_message":
		var req SDKControlMcpMessageRequest
		err := json.Unmarshal(data, &req)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse mcp_message control request",
				err,
			).WithMessageType("mcp_message")
		}

		return req, nil
	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf(
				"unsupported control request subtype: %s",
				envelope.Subtype,
			),
			nil,
		).WithMessageType(envelope.Subtype)
	}
}

// SDKControlResponse represents control responses received FROM the Claude CLI.
// These are responses to control requests sent by the SDK.
type SDKControlResponse struct {
	BaseMessage
	Response ControlResponseVariant `json:"response"`
}

func (SDKControlResponse) Type() string { return "control_response" }

// ControlResponseVariant is the interface for all control response variants.
type ControlResponseVariant interface {
	// Subtype returns the control response variant's subtype.
	Subtype() string
	// RequestID returns the request id of the control response variant.
	RequestID() string
	controlResponseVariant()
}

// ControlSuccessResponse represents a successful control response.
type ControlSuccessResponse struct {
	SubtypeField   string               `json:"subtype"` // "success"
	RequestIDField string               `json:"request_id"`
	Response       map[string]JSONValue `json:"response,omitempty"`
}

func (ControlSuccessResponse) Subtype() string {
	return "success"
}
func (r ControlSuccessResponse) RequestID() string {
	return r.RequestIDField
}
func (r ControlSuccessResponse) controlResponseVariant() {}

// ControlErrorResponse represents a failed control response.
type ControlErrorResponse struct {
	SubtypeField   string `json:"subtype"` // "error"
	RequestIDField string `json:"request_id"`
	Error          string `json:"error"`
}

func (r ControlErrorResponse) Subtype() string {
	return r.SubtypeField
}
func (r ControlErrorResponse) RequestID() string {
	return r.RequestIDField
}
func (ControlErrorResponse) controlResponseVariant() {}

// UnmarshalJSON custom unmarshaler for SDKControlResponse to handle
// the response variant.
func (r *SDKControlResponse) UnmarshalJSON(data []byte) error {
	type Alias struct {
		BaseMessage
		Response json.RawMessage `json:"response"`
	}

	var aux Alias
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse SDKControlResponse JSON",
			err,
		).WithMessageType("control_response").WithSessionID(aux.SessionIDField)
	}

	r.BaseMessage = aux.BaseMessage

	// Decode the response variant
	variant, err := decodeControlResponseVariant(aux.Response)
	if err != nil {
		return clauderrs.NewProtocolError(
			clauderrs.ErrCodeInvalidMessage,
			"failed to decode control response variant",
			err,
		).WithMessageType("control_response").WithSessionID(aux.SessionIDField)
	}
	r.Response = variant

	return nil
}

// decodeControlResponseVariant converts raw JSON into a typed control
// response variant.
func decodeControlResponseVariant(data []byte) (ControlResponseVariant, error) {
	var envelope struct {
		Subtype string `json:"subtype"`
	}
	err := json.Unmarshal(data, &envelope)
	if err != nil {
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeMessageParseFailed,
			"failed to parse control response subtype envelope",
			err,
		)
	}

	switch envelope.Subtype {
	case "success":
		var resp ControlSuccessResponse
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse success control response",
				err,
			).WithMessageType("success")
		}

		return resp, nil
	case "error":
		var resp ControlErrorResponse
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return nil, clauderrs.NewProtocolError(
				clauderrs.ErrCodeMessageParseFailed,
				"failed to parse error control response",
				err,
			).WithMessageType("error")
		}

		return resp, nil
	default:
		return nil, clauderrs.NewProtocolError(
			clauderrs.ErrCodeUnknownMessageType,
			fmt.Sprintf(
				"unsupported control response subtype: %s",
				envelope.Subtype,
			),
			nil,
		).WithMessageType(envelope.Subtype)
	}
}

// SDKControlPermissionRequest represents permission requests FROM
// the Claude CLI.
// The CLI asks the SDK whether a tool should be allowed to execute.
// The SDK must respond with an SDKControlResponse.
type SDKControlPermissionRequest struct {
	BaseMessage
	RequestIDField        string               `json:"request_id"`
	SubtypeField string `json:"subtype"` // "can_use_tool"
	ToolName              string               `json:"tool_name"`
	Input                 map[string]JSONValue `json:"input"`
	PermissionSuggestions []JSONValue `json:"permission_suggestions,omitempty"`
	BlockedPath           *string              `json:"blocked_path,omitempty"`
}

func (SDKControlPermissionRequest) Type() string        { return ControlRequest }
func (r SDKControlPermissionRequest) Subtype() string   { return r.SubtypeField }
func (r SDKControlPermissionRequest) RequestID() string { return r.RequestIDField }

// SDKHookCallbackRequest represents hook callback requests FROM the Claude CLI.
// The CLI calls a registered hook and expects the SDK to respond
// with hook output.
// The SDK must respond with an SDKControlResponse.
type SDKHookCallbackRequest struct {
	BaseMessage
	RequestIDField string    `json:"request_id"`
	SubtypeField   string    `json:"subtype"` // "hook_callback"
	CallbackID     string    `json:"callback_id"`
	Input          JSONValue `json:"input"`
	ToolUseID      *string   `json:"tool_use_id,omitempty"`
}

func (SDKHookCallbackRequest) Type() string        { return ControlRequest }
func (r SDKHookCallbackRequest) Subtype() string   { return r.SubtypeField }
func (r SDKHookCallbackRequest) RequestID() string { return r.RequestIDField }
