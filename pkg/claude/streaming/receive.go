package streaming

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// ReceiveMessages streams messages from Claude CLI.
// It reads from the internal message channels, parses messages using
// the parser, and forwards them to the returned channels.
// Returns channels for messages and errors that close when stream ends.
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go func() {
		defer close(msgOutCh)
		defer close(errOutCh)

		if err := s.receiveLoop(ctx, msgOutCh, errOutCh); err != nil {
			errOutCh <- err
		}
	}()

	return msgOutCh, errOutCh
}

// receiveLoop runs the message receiving loop.
func (s *Service) receiveLoop(
	ctx context.Context,
	msgOutCh chan messages.Message,
	_ chan error,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case msg, ok := <-s.msgCh:
			if !ok {
				return nil
			}
			if err := s.handleMessage(msg, msgOutCh); err != nil {
				return err
			}

		case err := <-s.errCh:
			if err != nil {
				return err
			}
		}
	}
}

// handleMessage parses and sends a message to the output channel.
func (s *Service) handleMessage(
	msg map[string]any,
	msgOutCh chan messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(msg)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}
	msgOutCh <- parsedMsg

	return nil
}
