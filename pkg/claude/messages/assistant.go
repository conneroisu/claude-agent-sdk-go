package messages

// AssistantMessage represents Claude's response.
// Assistant messages contain the model's output as a series of content blocks,
// which can include text, thinking, tool uses, and tool results.
type AssistantMessage struct {
	// Content is a list of content blocks in the response
	Content []ContentBlock

	// StopReason indicates why generation stopped
	// (e.g., "end_turn", "max_tokens")
	StopReason *string

	// Model is the identifier of the model that generated this response
	Model *string
}

func (*AssistantMessage) message() {}
