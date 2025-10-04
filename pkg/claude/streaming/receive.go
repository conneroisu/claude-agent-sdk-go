// Package streaming provides message receiving functionality.
package streaming

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// ReceiveMessages returns channels for receiving streaming messages.
// The returned channels provide async access to incoming messages and errors.
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	msgOutCh := make(chan messages.Message)
	errOutCh := make(chan error, 1)

	go func() {
		defer close(msgOutCh)
		defer close(errOutCh)

		s.receiveLoop(ctx, msgOutCh, errOutCh)
	}()

	return msgOutCh, errOutCh
}

// receiveLoop handles the message receive loop.
// Continuously reads messages from internal channels and forwards to output.
func (s *Service) receiveLoop(
	ctx context.Context,
	msgOutCh chan<- messages.Message,
	errOutCh chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-s.msgCh:
			if !ok {
				return
			}
			if err := s.parseAndSend(msg, msgOutCh); err != nil {
				errOutCh <- err

				return
			}

		case err := <-s.errCh:
			if err != nil {
				errOutCh <- err

				return
			}
		}
	}
}

// parseAndSend parses a message and sends it to the output channel.
func (s *Service) parseAndSend(
	msg map[string]any,
	msgOutCh chan<- messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(msg)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	msgOutCh <- parsedMsg

	return nil
}
