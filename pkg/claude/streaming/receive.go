package streaming

import (
	"context"
	"errors"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// ReceiveMessages returns channels for streaming messages and errors.
// Messages are parsed and forwarded through output channels.
// Error channel receives parsing errors or context cancellation.
// The same channels are returned on multiple calls.
// Channels are closed when the connection ends or Close() is called.
func (s *Service) ReceiveMessages(
	ctx context.Context,
) (<-chan messages.Message, <-chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	outMsgCh := make(chan messages.Message)
	outErrCh := make(chan error, 1)

	if !s.connected {
		outErrCh <- errors.New("not connected")
		close(outMsgCh)
		close(outErrCh)

		return outMsgCh, outErrCh
	}

	go s.streamMessages(ctx, outMsgCh, outErrCh)

	return outMsgCh, outErrCh
}

// streamMessages processes raw messages from router channels.
// Parses each message and forwards to output channels.
// Terminates on context cancellation, channel closure, or error.
func (s *Service) streamMessages(
	ctx context.Context,
	msgCh chan<- messages.Message,
	errCh chan<- error,
) {
	defer close(msgCh)
	defer close(errCh)

	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()

			return

		case rawMsg, ok := <-s.msgCh:
			if !ok {
				return
			}
			if err := s.processMessage(rawMsg, msgCh); err != nil {
				errCh <- err

				return
			}

		case err, ok := <-s.errCh:
			if ok && err != nil {
				errCh <- err
			}

			return
		}
	}
}

// processMessage parses a raw message map into a typed Message.
// Returns parsing error if message format is invalid.
func (s *Service) processMessage(
	raw map[string]any,
	msgCh chan<- messages.Message,
) error {
	msg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	msgCh <- msg

	return nil
}
