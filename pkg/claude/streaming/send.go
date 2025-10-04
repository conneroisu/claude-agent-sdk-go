package streaming

import (
	"context"
	"encoding/json"
	"fmt"
)

// SendMessage sends a user message to Claude CLI.
// It formats the message as a user prompt and writes it to the transport.
// Returns error if marshaling or writing fails.
func (s *Service) SendMessage(ctx context.Context, msg string) error {
	userMsg := map[string]any{
		"type":   "user",
		"prompt": msg,
	}

	msgBytes, err := json.Marshal(userMsg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := s.transport.Write(ctx, string(msgBytes)+"\n"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
