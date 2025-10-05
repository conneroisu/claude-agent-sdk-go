package messages

// UserMessage represents user input to Claude.
// User messages contain the input content and optionally reference a parent
// tool use ID for continuing multi-turn interactions.
type UserMessage struct {
	// Content is the user's input, either as a string or structured blocks
	Content MessageContent

	// ParentToolUseID optionally references a tool use from a previous turn
	ParentToolUseID *string
}

func (*UserMessage) message() {}
