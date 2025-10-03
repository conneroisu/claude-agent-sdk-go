//nolint:revive // comments-density: code is self-documenting
package streaming

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// ReceiveMessages returns a channel of messages from Claude.
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go func() {
		defer close(msgOutCh)
		defer close(errOutCh)

		if err := s.streamMessages(
			ctx,
			msgOutCh,
			errOutCh,
		); err != nil {
			errOutCh <- err
		}
	}()

	return msgOutCh, errOutCh
}

// streamMessages reads and routes messages.
func (s *Service) streamMessages(
	ctx context.Context,
//nolint:revive // unused-parameter: errOutCh required for interface
	msgOutCh chan<- messages.Message,
	errOutCh chan<- error,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-s.msgCh:
			if !ok {
				return nil
			}

			if err := s.handleMessage(
				msg,
				msgOutCh,
			); err != nil {
				return err
			}

		case err := <-s.errCh:
			if err != nil {
				return err
			}
		}
	}
}

// handleMessage parses and sends a message.
func (s *Service) handleMessage(
	raw map[string]any,
	msgOutCh chan<- messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	msgOutCh <- parsedMsg

	return nil
}
