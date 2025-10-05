package messages

// SystemMessage represents control messages from the CLI.
// System messages convey initialization state, compact boundaries,
// and other control information.
type SystemMessage struct {
	// Data contains the system message variant data
	Data SystemMessageData
}

func (*SystemMessage) message() {}

// SystemInitMessage represents session initialization.
// This message is sent when a new session begins or when resuming an
// existing session.
type SystemInitMessage struct {
	// SessionID uniquely identifies this conversation session
	SessionID string

	// IsResumed indicates whether this is a resumed session
	IsResumed bool

	// ForkedFrom optionally indicates the session this was forked from
	ForkedFrom *string
}

func (*SystemInitMessage) systemMessageData() {}

// SystemCompactBoundaryMessage marks a conversation compaction point.
// Compact boundaries indicate where conversation history was condensed
// to save context tokens.
type SystemCompactBoundaryMessage struct {
	// CompactID uniquely identifies this compaction operation
	CompactID string

	// TokensSaved indicates how many tokens were saved by compaction
	TokensSaved int
}

func (*SystemCompactBoundaryMessage) systemMessageData() {}

// SystemGenericMessage represents other system message types.
// This type allows for extensibility when the CLI introduces new
// system message variants.
type SystemGenericMessage struct {
	// Type identifies the system message subtype
	Type string

	// Data contains arbitrary system message data
	Data map[string]any
}

func (*SystemGenericMessage) systemMessageData() {}
