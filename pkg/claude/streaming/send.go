// Package streaming provides message sending functionality.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"
)

// SendMessage sends a message in the streaming conversation.
// Formats the message as a user message and writes to the transport.
func (s *Service) SendMessage(ctx context.Context, msg string) error {
	// Create user message format expected by Claude CLI
	userMsg := map[string]any{
		"type":   "user",
		"prompt": msg,
	}

	// Marshal to JSON
	msgBytes, err := json.Marshal(userMsg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Write to transport with newline delimiter
	if err := s.transport.Write(ctx, string(msgBytes)+"\n"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
