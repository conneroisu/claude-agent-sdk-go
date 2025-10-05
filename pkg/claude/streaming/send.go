package streaming

import (
	"context"
	"errors"
)

// SendMessage sends a user message to Claude in the active conversation.
// It returns an error if not connected or if the write fails.
func (s *Service) SendMessage(
	ctx context.Context,
	prompt string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return errors.New("not connected")
	}

	return s.sendPromptUnsafe(ctx, prompt)
}
